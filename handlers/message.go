package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"IM2.0/database"
	"IM2.0/hub"
	"IM2.0/middleware"
	"IM2.0/models"
	"IM2.0/utils"

	"github.com/gin-gonic/gin"
)

// SendMessage sends a private message
func SendMessage(c *gin.Context) {
	fromUserID := middleware.GetUserID(c)
	var req struct {
		ToUserID int    `json:"to_user_id"`
		Content  string `json:"content"`
		MsgType  int    `json:"msg_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}

	if req.ToUserID == 0 || req.Content == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "接收人或内容不能为空"})
		return
	}

	// Truncate content for weak network optimization
	content := utils.TruncateContent(req.Content, 5000)

	result, err := database.DB.Exec(
		"INSERT INTO messages (from_user_id, to_user_id, content, msg_type) VALUES (?, ?, ?, ?)",
		fromUserID, req.ToUserID, content, req.MsgType,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "发送失败"})
		return
	}

	msgID, _ := result.LastInsertId()

	// Check if recipient is online
	if hub.DefaultHub.IsOnline(req.ToUserID) {
		// Update status to delivered
		database.DB.Exec("UPDATE messages SET status = 1 WHERE id = ?", msgID)

		// Push via SSE
		var fromName string
		database.DB.QueryRow("SELECT nickname FROM users WHERE id = ?", fromUserID).Scan(&fromName)

		msgData, _ := json.Marshal(gin.H{
			"id":           msgID,
			"from_user_id": fromUserID,
			"from_name":    fromName,
			"content":      content,
			"msg_type":     req.MsgType,
			"created_at":   time.Now().Format("2006-01-02 15:04:05"),
		})
		hub.DefaultHub.PushEvent(req.ToUserID, "message", string(msgData))
	} else {
		// Store as offline message (weak network optimization)
		database.DB.Exec(
			"INSERT INTO offline_messages (user_id, message_id) VALUES (?, ?)",
			req.ToUserID, msgID,
		)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "发送成功",
		Data:    gin.H{"id": msgID, "status": 1},
	})
}

// GetMessages returns chat history between two users
func GetMessages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	otherIDStr := c.Param("id")
	otherID, _ := strconv.Atoi(otherIDStr)
	if otherID == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "无效的用户ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	beforeStr := c.Query("before")
	before := int64(0)
	if beforeStr != "" {
		before, _ = strconv.ParseInt(beforeStr, 10, 64)
	}

	var rows *sql.Rows
	var err error
	if before > 0 {
		rows, err = database.DB.Query(`
			SELECT m.id, m.from_user_id, m.to_user_id, m.content, m.msg_type, m.status, m.created_at,
			       fu.nickname as from_name, tu.nickname as to_name
			FROM messages m
			JOIN users fu ON m.from_user_id = fu.id
			JOIN users tu ON m.to_user_id = tu.id
			WHERE ((m.from_user_id = ? AND m.to_user_id = ?) OR (m.from_user_id = ? AND m.to_user_id = ?))
			  AND m.id < ?
			ORDER BY m.id DESC LIMIT ?`,
			userID, otherID, otherID, userID, before, limit)
	} else {
		rows, err = database.DB.Query(`
			SELECT m.id, m.from_user_id, m.to_user_id, m.content, m.msg_type, m.status, m.created_at,
			       fu.nickname as from_name, tu.nickname as to_name
			FROM messages m
			JOIN users fu ON m.from_user_id = fu.id
			JOIN users tu ON m.to_user_id = tu.id
			WHERE (m.from_user_id = ? AND m.to_user_id = ?) OR (m.from_user_id = ? AND m.to_user_id = ?)
			ORDER BY m.id DESC LIMIT ?`,
			userID, otherID, otherID, userID, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		rows.Scan(&m.ID, &m.FromUserID, &m.ToUserID, &m.Content, &m.MsgType, &m.Status, &m.CreatedAt,
			&m.FromName, &m.ToName)
		messages = append(messages, m)
	}

	// Reverse to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: messages})
}

// DeleteMessage deletes a message
func DeleteMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var fromUserID int
	err := database.DB.QueryRow("SELECT from_user_id FROM messages WHERE id = ?", id).Scan(&fromUserID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "消息不存在"})
		return
	}
	if fromUserID != userID {
		c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "只能删除自己的消息"})
		return
	}

	_, err = database.DB.Exec("DELETE FROM messages WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "删除失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已删除"})
}

// SendGroupMessage sends a group message
func SendGroupMessage(c *gin.Context) {
	fromUserID := middleware.GetUserID(c)
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	var req struct {
		Content string `json:"content"`
		MsgType int    `json:"msg_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "内容不能为空"})
		return
	}

	content := utils.TruncateContent(req.Content, 5000)

	result, err := database.DB.Exec(
		"INSERT INTO group_messages (group_id, from_user_id, content, msg_type) VALUES (?, ?, ?, ?)",
		groupID, fromUserID, content, req.MsgType,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "发送失败"})
		return
	}

	msgID, _ := result.LastInsertId()

	// Get all group members and push via SSE
	rows, err := database.DB.Query(
		"SELECT user_id FROM group_members WHERE group_id = ? AND user_id != ?",
		groupID, fromUserID,
	)
	if err == nil {
		var fromName string
		database.DB.QueryRow("SELECT nickname FROM users WHERE id = ?", fromUserID).Scan(&fromName)

		msgData, _ := json.Marshal(gin.H{
			"id":           msgID,
			"group_id":     groupID,
			"from_user_id": fromUserID,
			"from_name":    fromName,
			"content":      content,
			"msg_type":     req.MsgType,
			"created_at":   time.Now().Format("2006-01-02 15:04:05"),
		})
		for rows.Next() {
			var uid int
			rows.Scan(&uid)
			hub.DefaultHub.PushEvent(uid, "group_message", string(msgData))
		}
		rows.Close()
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "发送成功",
		Data:    gin.H{"id": msgID},
	})
}

// GetGroupMessages returns group chat history
func GetGroupMessages(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, _ := strconv.Atoi(groupIDStr)

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := database.DB.Query(`
		SELECT gm.id, gm.group_id, gm.from_user_id, gm.content, gm.msg_type, gm.created_at,
		       u.nickname as from_name
		FROM group_messages gm
		JOIN users u ON gm.from_user_id = u.id
		WHERE gm.group_id = ?
		ORDER BY gm.id DESC LIMIT ?`, groupID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var messages []models.GroupMessage
	for rows.Next() {
		var m models.GroupMessage
		rows.Scan(&m.ID, &m.GroupID, &m.FromUserID, &m.Content, &m.MsgType, &m.CreatedAt, &m.FromName)
		messages = append(messages, m)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: messages})
}

// SSEHandler handles Server-Sent Events connection
func SSEHandler(c *gin.Context) {
	userID := middleware.GetUserID(c)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Create client
	client := &hub.Client{
		UserID: userID,
		Chan:   make(chan []byte, 100),
	}
	hub.DefaultHub.Register(client)
	defer hub.DefaultHub.Unregister(client)

	// Send offline messages first (weak network optimization)
	rows, err := database.DB.Query(`
		SELECT m.id, m.from_user_id, m.content, m.msg_type, m.created_at, u.nickname
		FROM offline_messages om
		JOIN messages m ON om.message_id = m.id
		JOIN users u ON m.from_user_id = u.id
		WHERE om.user_id = ? AND om.delivered = 0
		ORDER BY m.id ASC`, userID)
	if err == nil {
		for rows.Next() {
			var id, fromID, msgType int
			var content, fromName, createdAt string
			rows.Scan(&id, &fromID, &content, &msgType, &createdAt, &fromName)
			msgData, _ := json.Marshal(gin.H{
				"id":           id,
				"from_user_id": fromID,
				"from_name":    fromName,
				"content":      content,
				"msg_type":     msgType,
				"created_at":   createdAt,
			})
			fmt.Fprintf(c.Writer, "event: offline_message\ndata: %s\n\n", string(msgData))
			c.Writer.Flush()
		}
		rows.Close()
		// Mark as delivered
		database.DB.Exec("UPDATE offline_messages SET delivered = 1 WHERE user_id = ?", userID)
	}

	// Heartbeat ticker for weak network keep-alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	notify := c.Writer.CloseNotify()

	for {
		select {
		case msg := <-client.Chan:
			c.Writer.Write(msg)
			c.Writer.Flush()
		case <-ticker.C:
			c.Writer.Write([]byte(": heartbeat\n\n"))
			c.Writer.Flush()
		case <-notify:
			return
		}
	}
}

// GetMessageStatus returns message delivery status
func GetMessageStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var status int
	err := database.DB.QueryRow("SELECT status FROM messages WHERE id = ?", id).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "消息不存在"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: gin.H{"status": status}})
}
