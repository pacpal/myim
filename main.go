package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"im/database"
	"im/handlers"
	"im/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (ignored if not found, e.g. in production)
	_ = godotenv.Load()

	// Database configuration - reads from env, falls back to local defaults
	dbConfig := database.Config{
		Host:     getEnv("DB_HOST", "127.0.0.1"),
		Port:     getEnvInt("DB_PORT", 3307),
		User:     getEnv("DB_USER", "root"),
		Password: getEnv("DB_PASSWORD", "root"),
		DBName:   getEnv("DB_NAME", "im_system"),
	}

	if err := database.InitDB(dbConfig); err != nil {
		log.Fatalf("数据库连接失败: %v\n请确保MySQL已启动且im_system数据库已创建(执行database/schema.sql)", err)
	}
	defer database.CloseDB()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	// Rate limiter for DoS defense (60 requests per minute per IP)
	globalLimiter := middleware.NewRateLimiter(120, time.Minute)
	r.Use(middleware.RateLimitMiddleware(globalLimiter))

	// Stricter rate limiter for login endpoint
	loginLimiter := middleware.NewRateLimiter(10, time.Minute)

	// Serve static frontend files
	r.Static("/static", "./frontend")
	r.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	// API routes
	api := r.Group("/api")
	{
		// Auth routes
		api.POST("/register", handlers.Register)
		api.POST("/login", middleware.RateLimitMiddleware(loginLimiter), handlers.Login)

		// Attack demonstration routes (no auth required for demo)
		attack := api.Group("/attack")
		{
			attack.GET("/sql/vulnerable", handlers.SQLInjectionVulnerable)
			attack.GET("/sql/safe", handlers.SQLInjectionSafe)
			attack.POST("/xss/vulnerable", handlers.XSSVulnerable)
			attack.GET("/xss/vulnerable", handlers.XSSVulnerable)
			attack.POST("/xss/safe", handlers.XSSSafe)
			attack.GET("/xss/safe", handlers.XSSSafe)
			attack.GET("/dos/vulnerable", handlers.DoSVulnerable)
			attack.POST("/tamper/:id", handlers.TamperMessage)
			attack.GET("/guide", handlers.GetAttackGuide)
		}

		// Protected routes (require auth)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/ping", handlers.Ping)
			protected.POST("/logout", handlers.Logout)
			protected.GET("/profile", handlers.GetProfile)
			protected.PUT("/profile", handlers.UpdateProfile)

			// Friend routes
			protected.GET("/friends", handlers.GetFriends)
			protected.POST("/friends/request", handlers.SendFriendRequest)
			protected.GET("/friends/requests", handlers.GetFriendRequests)
			protected.PUT("/friends/requests/:id/accept", handlers.AcceptFriendRequest)
			protected.PUT("/friends/requests/:id/reject", handlers.RejectFriendRequest)
			protected.DELETE("/friends/:id", handlers.DeleteFriend)
			protected.GET("/users/search", handlers.SearchUser)

			// Group routes
			protected.GET("/groups", handlers.GetGroups)
			protected.POST("/groups", handlers.CreateGroup)
			protected.GET("/groups/:id/members", handlers.GetGroupMembers)
			protected.POST("/groups/:id/members", handlers.AddGroupMember)
			protected.DELETE("/groups/:id/members", handlers.RemoveGroupMember)
			protected.DELETE("/groups/:id", handlers.DeleteGroup)

			// Message routes
			protected.POST("/messages", handlers.SendMessage)
			protected.GET("/messages/:id", handlers.GetMessages)
			protected.DELETE("/messages/:id", handlers.DeleteMessage)
			protected.GET("/messages/:id/status", handlers.GetMessageStatus)
			protected.POST("/groups/:id/messages", handlers.SendGroupMessage)
			protected.GET("/groups/:id/messages", handlers.GetGroupMessages)

			// SSE endpoint for real-time push
			protected.GET("/sse", handlers.SSEHandler)

			// Security audit
			protected.GET("/logs/login", handlers.GetLoginLogs)

			// Integrity check (any logged-in user can verify)
			protected.GET("/integrity/check", handlers.IntegrityCheck)

			// Admin routes (audit center, admin only)
			admin := protected.Group("/admin", middleware.AdminMiddleware())
			{
				admin.GET("/audit/logs", handlers.GetAuditLogs)
				admin.GET("/integrity/alerts", handlers.GetIntegrityAlerts)
				admin.PUT("/integrity/alerts/:id/resolve", handlers.ResolveIntegrityAlert)
			}
		}
	}

	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)
	log.Printf("IM系统启动成功，访问 http://localhost:%s", port)
	log.Printf("前端页面: http://localhost:%s/", port)
	log.Printf("API文档: http://localhost:%s/api/attack/guide", port)

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return defaultValue
}
