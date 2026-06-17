package handlers

import (
	"im/database"
	"im/util"
)

// RecordAudit appends a tamper-proof audit log entry (hash-chained).
// Call this for every sensitive operation (login, send_msg, tamper, integrity_check...).
func RecordAudit(actorID int, actorName, action, detail, ip string) {
	// Get the previous audit log's hash to chain from
	var prevHash string
	database.DB.QueryRow(
		"SELECT curr_hash FROM audit_logs ORDER BY id DESC LIMIT 1",
	).Scan(&prevHash)

	// Insert first with placeholder created_at, then update hash with real created_at.
	// Simpler: use NOW() via a two-step: insert then read id+created_at then update hash.
	res, err := database.DB.Exec(
		`INSERT INTO audit_logs (actor_id, actor_name, action, detail, ip_address, prev_hash)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		actorID, actorName, action, detail, ip, prevHash,
	)
	if err != nil {
		return
	}
	id, _ := res.LastInsertId()
	var createdAt string
	database.DB.QueryRow("SELECT created_at FROM audit_logs WHERE id = ?", id).Scan(&createdAt)

	currHash := util.ComputeAuditHash(prevHash, actorID, action, detail, createdAt)
	database.DB.Exec("UPDATE audit_logs SET curr_hash = ? WHERE id = ?", currHash, id)
}
