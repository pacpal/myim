package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"im/database"
	"im/middleware"
	"im/models"
	"im/utils"

	"github.com/gin-gonic/gin"
)

// SearchGroups searches groups by name (for joining)
func SearchGroups(c *gin.Context) {
	userID := middleware.GetUserID(c)
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: []gin.H{}})
		return
	}

	rows, err := database.DB.Query(`
		SELECT g.id, g.name, g.description, g.created_at, u.nickname as owner_name
		FROM `+"`groups`"+` g
		JOIN users u ON g.owner_id = u.id
		WHERE g.name LIKE ?
		ORDER BY g.id DESC LIMIT 50`, "%"+utils.SanitizeInput(keyword)+"%")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	myGroups := make(map[int]bool)
	myRows, _ := database.DB.Query("SELECT group_id FROM group_members WHERE user_id = ?", userID)
	if myRows != nil {
		for myRows.Next() {
			var gid int
			myRows.Scan(&gid)
			myGroups[gid] = true
		}
		myRows.Close()
	}

	var groups []gin.H
	for rows.Next() {
		var id int
		var name, description, createdAt, ownerName string
		rows.Scan(&id, &name, &description, &createdAt, &ownerName)
		joined, _ := myGroups[id]
		groups = append(groups, gin.H{
			"id":          id,
			"name":        name,
			"description": description,
			"owner_name":  ownerName,
			"joined":      joined,
		})
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: groups})
}

// JoinGroup lets a user join an existing group
func JoinGroup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)
	if groupID == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的群ID"})
		return
	}

	// 检查群是否存在
	var exists int
	database.DB.QueryRow("SELECT 1 FROM `groups` WHERE id = ?", groupID).Scan(&exists)
	if exists == 0 {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "群不存在"})
		return
	}

	// 检查是否已在群中
	var alreadyIn int
	database.DB.QueryRow("SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID).Scan(&alreadyIn)
	if alreadyIn > 0 {
		c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已在群中"})
		return
	}

	_, err := database.DB.Exec("INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 0)", groupID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "加入失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "加入成功"})
}

// CreateGroup creates a new group
func CreateGroup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	name := utils.SanitizeInput(req.Name)
	description := utils.SanitizeInput(req.Description)
	if name == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "群名不能为空"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "事务失败"})
		return
	}

	result, err := tx.Exec(
		"INSERT INTO `groups` (name, owner_id, description) VALUES (?, ?, ?)",
		name, userID, description,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "创建失败"})
		return
	}

	groupID, _ := result.LastInsertId()
	_, err = tx.Exec(
		"INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 2)",
		groupID, userID,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "添加群主失败"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "创建成功",
		Data:    gin.H{"id": groupID, "name": name},
	})
}

// GetGroups returns groups the user belongs to
func GetGroups(c *gin.Context) {
	userID := middleware.GetUserID(c)
	rows, err := database.DB.Query(`
		SELECT g.id, g.name, g.owner_id, g.avatar, g.description, g.created_at,
		       u.nickname as owner_name
		FROM group_members gm
		JOIN `+"`groups`"+` g ON gm.group_id = g.id
		JOIN users u ON g.owner_id = u.id
		WHERE gm.user_id = ?
		ORDER BY g.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.Avatar, &g.Description, &g.CreatedAt, &g.OwnerName)
		groups = append(groups, g)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: groups})
}

// GetGroupMembers returns members of a group
func GetGroupMembers(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	rows, err := database.DB.Query(`
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.joined_at,
		       u.username, u.nickname, u.avatar
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ?
		ORDER BY gm.role DESC, gm.joined_at ASC`, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var members []models.GroupMember
	for rows.Next() {
		var m models.GroupMember
		rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.Role, &m.JoinedAt,
			&m.Username, &m.Nickname, &m.Avatar)
		members = append(members, m)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: members})
}

// AddGroupMember adds a member to a group
func AddGroupMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	var req struct {
		UserID int `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	// Check if requester is group owner/admin
	var role int
	err := database.DB.QueryRow(
		"SELECT role FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows || role < 1 {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权限"})
		return
	}

	_, err = database.DB.Exec(
		"INSERT INTO group_members (group_id, user_id, role) VALUES (?, ?, 0)",
		groupID, req.UserID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "添加失败，可能已在群中"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已加入群聊"})
}

// RemoveGroupMember removes a member from group
func RemoveGroupMember(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	memberIDStr := c.Query("user_id")
	memberID, _ := strconv.Atoi(memberIDStr)

	// Check permission
	var role int
	err := database.DB.QueryRow(
		"SELECT role FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID, userID,
	).Scan(&role)
	if err == sql.ErrNoRows || role < 1 {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "无权限"})
		return
	}

	_, err = database.DB.Exec(
		"DELETE FROM group_members WHERE group_id = ? AND user_id = ?",
		groupID, memberID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "移除失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已移除成员"})
}

// DeleteGroup deletes a group (owner only)
func DeleteGroup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	var ownerID int
	err := database.DB.QueryRow("SELECT owner_id FROM `groups` WHERE id = ?", groupID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "群不存在"})
		return
	}
	if ownerID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "仅群主可解散群"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM `groups` WHERE id = ?", groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "解散失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "群已解散"})
}
