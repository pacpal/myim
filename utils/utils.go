package utils

import (
	"crypto/rand"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// GenerateToken generates a random hex token for sessions
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// EscapeHTML escapes HTML special characters to prevent XSS
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// SanitizeInput removes potentially dangerous content
func SanitizeInput(s string) string {
	s = strings.TrimSpace(s)
	s = html.EscapeString(s)
	return s
}

// ValidateUsername checks if username is valid
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	return matched
}

// ValidatePassword checks password strength
func ValidatePassword(password string) bool {
	return utf8.RuneCountInString(password) >= 6
}

// TruncateContent limits message content length for weak network
func TruncateContent(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	r := []rune(s)
	return string(r[:maxLen])
}
