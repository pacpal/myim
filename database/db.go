package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// InitDB opens the database connection and retries until the server is reachable.
// This makes the app resilient to the MySQL container not being ready yet
// (typical in docker-compose based deployments).
func InitDB(cfg Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)

	// Retry ping so that the app does not crash when MySQL is still starting.
	maxRetries := getEnvInt("DB_MAX_RETRIES", 30)
	retryInterval := getEnvDuration("DB_RETRY_INTERVAL", 2*time.Second)

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err = DB.Ping(); err == nil {
			break
		}
		lastErr = err
		log.Printf("Waiting for MySQL at %s:%d (%d/%d): %v", cfg.Host, cfg.Port, i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}
	if err != nil {
		return fmt.Errorf("failed to ping database after %d retries: %v (last error: %v)", maxRetries, err, lastErr)
	}

	log.Println("Database connected successfully")

	// Seed default admin account for the audit center (password: admin123)
	if err := seedAdmin(); err != nil {
		log.Printf("seed admin warning: %v", err)
	}

	return nil
}

// seedAdmin creates the default admin account if it does not exist
func seedAdmin() error {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = DB.Exec(
		"INSERT INTO users (username, password, nickname, role, status) VALUES (?, ?, ?, 1, 0)",
		"admin", string(hashed), "审计管理员",
	)
	if err != nil {
		return err
	}
	log.Println("Default admin account created (username: admin, password: admin123)")
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return defaultValue
}
