package handlers

import (
	"database/sql"
	"net/http"

	"im/database"
	"im/models"
	"im/utils"

	"github.com/gin-gonic/gin"
)

// SQLInjectionVulnerable demonstrates SQL injection attack (VULNERABLE - for demo only)
// This endpoint uses string concatenation - DO NOT use in production
func SQLInjectionVulnerable(c *gin.Context) {
	username := c.Query("username")

	// VULNERABLE: string concatenation allows SQL injection
	query := "SELECT id, username, nickname FROM users WHERE username = '" + username + "'"
	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusOK, models.APIResponse{
			Code:    500,
			Message: "SQL错误: " + err.Error(),
			Data:    gin.H{"query": query},
		})
		return
	}
	defer rows.Close()

	type User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Nickname)
		users = append(users, u)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "漏洞接口(仅演示)",
		Data:    gin.H{"query": query, "results": users},
	})
}

// SQLInjectionSafe demonstrates SQL injection defense (parameterized query)
func SQLInjectionSafe(c *gin.Context) {
	username := c.Query("username")

	// SAFE: parameterized query prevents SQL injection
	rows, err := database.DB.Query(
		"SELECT id, username, nickname FROM users WHERE username = ?",
		username,
	)
	if err != nil {
		c.JSON(http.StatusOK, models.APIResponse{
			Code:    500,
			Message: "SQL错误: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	type User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Nickname)
		users = append(users, u)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "防御接口(参数化查询)",
		Data:    gin.H{"results": users},
	})
}

// XSSVulnerable demonstrates XSS attack (VULNERABLE - returns raw content)
func XSSVulnerable(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Content = c.Query("content")
	}

	// VULNERABLE: returns raw content without escaping
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "漏洞接口(仅演示)",
		Data:    gin.H{"raw_content": req.Content},
	})
}

// XSSSafe demonstrates XSS defense (HTML escaping)
func XSSSafe(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Content = c.Query("content")
	}

	// SAFE: HTML escape the content
	escaped := utils.EscapeHTML(req.Content)

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "防御接口(HTML转义)",
		Data:    gin.H{"escaped_content": escaped, "raw_content": req.Content},
	})
}

// DoSVulnerable demonstrates DoS attack (no rate limit, heavy computation)
func DoSVulnerable(c *gin.Context) {
	// VULNERABLE: no rate limit, performs expensive operation
	// Simulate heavy computation
	result := 0
	for i := 0; i < 10000000; i++ {
		result += i
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "漏洞接口(无限制)",
		Data:    gin.H{"computation": result, "note": "此接口无任何限制，可被DoS攻击"},
	})
}

// GetLoginLogs returns recent login logs (for security audit)
func GetLoginLogs(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, user_id, username, ip_address, user_agent, status, created_at
		FROM login_logs
		ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var logs []models.LoginLog
	for rows.Next() {
		var l models.LoginLog
		var userID sql.NullInt64
		rows.Scan(&l.ID, &userID, &l.Username, &l.IPAddress, &l.UserAgent, &l.Status, &l.CreatedAt)
		if userID.Valid {
			l.UserID = int(userID.Int64)
		}
		logs = append(logs, l)
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: logs})
}

// GetAttackGuide returns attack/defense demonstration guide
func GetAttackGuide(c *gin.Context) {
	guide := []gin.H{
		{
			"type":        "SQL注入",
			"attack_url":  "/api/attack/sql/vulnerable?username=' OR '1'='1",
			"defense_url": "/api/attack/sql/safe?username=' OR '1'='1",
			"description": "漏洞接口使用字符串拼接SQL，攻击者可通过 ' OR '1'='1 绕过认证；防御接口使用参数化查询",
		},
		{
			"type":        "XSS跨站脚本",
			"attack_url":  "/api/attack/xss/vulnerable",
			"defense_url": "/api/attack/xss/safe",
			"method":      "POST",
			"body":        `{"content":"<script>alert('XSS')</script>"}`,
			"description": "漏洞接口直接返回原始内容，前端渲染会执行脚本；防御接口对HTML特殊字符转义",
		},
		{
			"type":        "DoS拒绝服务",
			"attack_url":  "/api/attack/dos/vulnerable",
			"defense_url": "/api/protected/ping",
			"description": "漏洞接口无限制执行重计算；防御接口通过速率限制中间件拦截高频请求",
		},
		{
			"type":        "数据篡改(创新点)",
			"attack_url":  "/api/attack/tamper/{id} (POST {\"content\":\"被篡改的内容\"})",
			"defense_url": "/api/integrity/check",
			"description": "攻击者直接修改数据库消息内容但不更新哈希链；防御端通过SHA256哈希链重新计算并比对，精确定位被篡改的记录。审计日志自身也受哈希链保护，防止管理员删日志",
		},
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: guide})
}

// Ping is a simple protected endpoint to test rate limiting
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "pong",
	})
}
