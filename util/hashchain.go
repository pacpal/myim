package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// ComputeHash computes the hash chain value for a message/audit record.
// Formula: SHA256(prev_hash || content || fromUserID || createdAt)
func ComputeHash(prevHash, content string, fromUserID int, createdAt string) string {
	data := prevHash + content + strconv.Itoa(fromUserID) + createdAt
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// ComputeAuditHash computes the hash for an audit log entry.
// Formula: SHA256(prev_hash || actor_id || action || detail || created_at)
func ComputeAuditHash(prevHash string, actorID int, action, detail, createdAt string) string {
	data := prevHash + strconv.Itoa(actorID) + action + detail + createdAt
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}
