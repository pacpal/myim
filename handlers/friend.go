package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"IM2.0/database"
	"IM2.0/hub"
	"IM2.0/middleware"
	"IM2.0/models"
	"IM2.0/utils"

	"github.com/gin-gonic/gin"
)

// SendFriendRequest sends a friend request
func SendFriendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		ToUserID int    `json:"to_user_id"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if req.ToUserID == userID {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "不能添加自己为好友"})
		return
	}

	message := utils.SanitizeInput(req.Message)

	// Check if already friends
	var count int
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM friends WHERE user_id = ? AND friend_id = ?",
		userID, req.ToUserID,
	).Scan(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "已经是好友"})
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO friend_requests (from_user_id, to_user_id, message) VALUES (?, ?, ?)",
		userID, req.ToUserID, message,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "发送失败"})
		return
	}

	id, _ := result.LastInsertId()

	// Push real-time notification via SSE
	if hub.DefaultHub.IsOnline(req.ToUserID) {
		var fromUser models.User
		database.DB.QueryRow("SELECT username, nickname FROM users WHERE id = ?", userID).
			Scan(&fromUser.Username, &fromUser.Nickname)
		hub.DefaultHub.PushEvent(req.ToUserID, "friend_request",
			`{"id":`+strconv.FormatInt(id, 10)+`,"from_user_id":`+strconv.Itoa(userID)+
				`,"from_username":"`+fromUser.Username+`","from_nickname":"`+fromUser.Nickname+`","message":"`+message+`"}`)
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "好友请求已发送"})
}

// GetFriendRequests returns pending friend requests
func GetFriendRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rows, err := database.DB.Query(`
		SELECT fr.id, fr.from_user_id, fr.message, fr.status, fr.created_at,
		       u.username, u.nickname
		FROM friend_requests fr
		JOIN users u ON fr.from_user_id = u.id
		WHERE fr.to_user_id = ? AND fr.status = 0
		ORDER BY fr.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var requests []models.FriendRequest
	for rows.Next() {
		var r models.FriendRequest
		rows.Scan(&r.ID, &r.FromUserID, &r.Message, &r.Status, &r.CreatedAt,
			&r.FromUsername, &r.FromNickname)
		requests = append(requests, r)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: requests})
}

// AcceptFriendRequest accepts a friend request
func AcceptFriendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的ID"})
		return
	}

	var fromUserID int
	err = database.DB.QueryRow(
		"SELECT from_user_id FROM friend_requests WHERE id = ? AND to_user_id = ? AND status = 0",
		id, userID,
	).Scan(&fromUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "请求不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "服务器错误"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "事务失败"})
		return
	}

	// Update request status
	_, err = tx.Exec("UPDATE friend_requests SET status = 1 WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}

	// Add bidirectional friendship
	_, err = tx.Exec("INSERT INTO friends (user_id, friend_id) VALUES (?, ?), (?, ?)",
		userID, fromUserID, fromUserID, userID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "添加好友失败"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "提交失败"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已添加好友"})
}

// RejectFriendRequest rejects a friend request
func RejectFriendRequest(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	_, err := database.DB.Exec(
		"UPDATE friend_requests SET status = 2 WHERE id = ? AND to_user_id = ?",
		id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "操作失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已拒绝"})
}

// GetFriends returns the friend list
func GetFriends(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rows, err := database.DB.Query(`
		SELECT f.id, f.friend_id, f.remark, f.created_at,
		       u.username, u.nickname, u.avatar, u.status
		FROM friends f
		JOIN users u ON f.friend_id = u.id
		WHERE f.user_id = ?
		ORDER BY u.status DESC, u.nickname ASC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var friends []models.Friend
	for rows.Next() {
		var f models.Friend
		rows.Scan(&f.ID, &f.FriendID, &f.Remark, &f.CreatedAt,
			&f.Username, &f.Nickname, &f.Avatar, &f.Status)
		friends = append(friends, f)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: friends})
}

// DeleteFriend removes a friend
func DeleteFriend(c *gin.Context) {
	userID := middleware.GetUserID(c)
	friendIDStr := c.Param("id")
	friendID, _ := strconv.Atoi(friendIDStr)

	tx, _ := database.DB.Begin()
	tx.Exec("DELETE FROM friends WHERE user_id = ? AND friend_id = ?", userID, friendID)
	tx.Exec("DELETE FROM friends WHERE user_id = ? AND friend_id = ?", friendID, userID)
	tx.Commit()

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已删除好友"})
}

// SearchUser searches users by username
func SearchUser(c *gin.Context) {
	keyword := c.Query("keyword")
	userID := middleware.GetUserID(c)
	if keyword == "" {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: []interface{}{}})
		return
	}

	// Parameterized query with LIKE - SQL injection defense
	rows, err := database.DB.Query(
		"SELECT id, username, nickname, avatar FROM users WHERE username LIKE ? AND id != ? LIMIT 20",
		"%"+keyword+"%", userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "搜索失败"})
		return
	}
	defer rows.Close()

	type UserResult struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	var users []UserResult
	for rows.Next() {
		var u UserResult
		rows.Scan(&u.ID, &u.Username, &u.Nickname, &u.Avatar)
		users = append(users, u)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: users})
}
