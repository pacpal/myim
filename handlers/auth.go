package handlers

import (
	"database/sql"
	"net/http"

	"IM2.0/database"
	"IM2.0/middleware"
	"IM2.0/models"
	"IM2.0/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Register handles user registration
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	if !utils.ValidateUsername(req.Username) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "用户名需3-50位字母数字下划线"})
		return
	}
	if !utils.ValidatePassword(req.Password) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "密码至少6位"})
		return
	}
	if req.Nickname == "" {
		req.Nickname = req.Username
	}

	// Check if username exists (parameterized query - SQL injection defense)
	var exists int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "服务器错误"})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusConflict, models.APIResponse{Code: 409, Message: "用户名已存在"})
		return
	}

	// Hash password with bcrypt
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "密码加密失败"})
		return
	}

	// Insert user (parameterized query)
	result, err := database.DB.Exec(
		"INSERT INTO users (username, password, nickname) VALUES (?, ?, ?)",
		req.Username, string(hashed), req.Nickname,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "注册失败"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "注册成功",
		Data:    gin.H{"id": id, "username": req.Username, "nickname": req.Nickname},
	})
}

// Login handles user login
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请求参数错误"})
		return
	}

	// Log login attempt
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	defer func() {
		database.DB.Exec(
			"INSERT INTO login_logs (user_id, username, ip_address, user_agent, status) VALUES (?, ?, ?, ?, ?)",
			nil, req.Username, ip, ua, 0,
		)
	}()

	// Parameterized query - SQL injection defense
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, password, nickname, avatar FROM users WHERE username = ?",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Nickname, &user.Avatar)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "用户名或密码错误"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "服务器错误"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "用户名或密码错误"})
		return
	}

	// Update login time and status
	database.DB.Exec("UPDATE users SET status = 1, last_login = NOW() WHERE id = ?", user.ID)

	// Log successful login
	database.DB.Exec(
		"INSERT INTO login_logs (user_id, username, ip_address, user_agent, status) VALUES (?, ?, ?, ?, ?)",
		user.ID, req.Username, ip, ua, 1,
	)

	// Create session
	token := utils.GenerateToken()
	middleware.Sessions.Set(token, user.ID)

	c.SetCookie("session_token", token, 3600*24, "/", "", false, true)

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "登录成功",
		Data: gin.H{
			"token":    token,
			"user_id":  user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
		},
	})
}

// Logout handles user logout
func Logout(c *gin.Context) {
	token := middleware.GetToken(c)
	if token != "" {
		middleware.Sessions.Delete(token)
	}
	userID := middleware.GetUserID(c)
	database.DB.Exec("UPDATE users SET status = 0 WHERE id = ?", userID)
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已退出登录"})
}

// GetProfile returns current user profile
func GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, username, nickname, avatar, status, last_login, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Username, &user.Nickname, &user.Avatar, &user.Status, &user.LastLogin, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: user})
}

// UpdateProfile updates user profile
func UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "参数错误"})
		return
	}
	nickname := utils.SanitizeInput(req.Nickname)
	avatar := utils.SanitizeInput(req.Avatar)

	_, err := database.DB.Exec(
		"UPDATE users SET nickname = ?, avatar = ? WHERE id = ?",
		nickname, avatar, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "更新成功"})
}
