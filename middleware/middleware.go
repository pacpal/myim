package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"im/database"
	"im/models"

	"github.com/gin-gonic/gin"
)

// SessionManager manages user sessions in memory
type SessionManager struct {
	sessions map[string]int
	mu       sync.RWMutex
}

var Sessions = &SessionManager{
	sessions: make(map[string]int),
}

func (sm *SessionManager) Set(token string, userID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[token] = userID
}

func (sm *SessionManager) Get(token string) (int, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	id, ok := sm.sessions[token]
	return id, ok
}

func (sm *SessionManager) Delete(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, token)
}

// AuthMiddleware validates session token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token, _ = c.Cookie("session_token")
		}
		token = strings.TrimPrefix(token, "Bearer ")

		// SSE/EventSource cannot set headers, allow token via query param
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Code:    401,
				Message: "未登录或登录已过期",
			})
			c.Abort()
			return
		}

		userID, ok := Sessions.Get(token)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Code:    401,
				Message: "无效的会话",
			})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("token", token)
		c.Next()
	}
}

// CORSMiddleware handles cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// RateLimiter implements a simple IP-based rate limiter for DoS defense
type visitor struct {
	count    int
	lastSeen time.Time
}

type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     int
	window   time.Duration
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
		return true
	}
	if time.Since(v.lastSeen) > rl.window {
		v.count = 1
		v.lastSeen = time.Now()
		return true
	}
	if v.count >= rl.rate {
		return false
	}
	v.count++
	v.lastSeen = time.Now()
	return true
}

// RateLimitMiddleware limits requests per IP for DoS defense
func RateLimitMiddleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, models.APIResponse{
				Code:    429,
				Message: "请求过于频繁，请稍后再试 (DoS防御已触发)",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
// Note: The security demo page (security.html) uses a relaxed CSP that allows
// 'unsafe-inline' so that the XSS vulnerability can actually be triggered for
// demonstration. All other pages keep a strict CSP (script-src 'self') which
// itself is an XSS defense mechanism.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// Relaxed CSP for the security demo page so XSS payloads can execute
		if strings.HasPrefix(c.Request.URL.Path, "/static/security.html") {
			c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: x")
		} else {
			c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		}
		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) int {
	if id, exists := c.Get("userID"); exists {
		return id.(int)
	}
	return 0
}

// GetToken extracts token from context
func GetToken(c *gin.Context) string {
	if token, exists := c.Get("token"); exists {
		return token.(string)
	}
	return ""
}

// AdminMiddleware requires the user to be an admin (role=1)
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, models.APIResponse{Code: 401, Message: "未登录"})
			c.Abort()
			return
		}
		var role int
		database.DB.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if role != 1 {
			c.JSON(http.StatusForbidden, models.APIResponse{Code: 403, Message: "需要管理员权限"})
			c.Abort()
			return
		}
		c.Set("role", role)
		c.Next()
	}
}
