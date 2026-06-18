package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"im/database"
	"im/hub"
	"im/middleware"
	"im/models"
	"im/util"

	"github.com/gin-gonic/gin"
)

// IntegrityCheck scans the entire hash chain and locates any tampered records.
// This is the core defense: even if a DBA directly modifies the database,
// the recomputed hash won't match the stored curr_hash.
func IntegrityCheck(c *gin.Context) {
	var alerts []gin.H

	// 1. Check private messages chain
	rows, err := database.DB.Query(
		`SELECT id, prev_hash, curr_hash, content, from_user_id, created_at
		 FROM messages ORDER BY id ASC`)
	if err == nil {
		expectedPrev := ""
		for rows.Next() {
			var id, fromUserID int
			var prevHash, currHash, content, createdAt string
			rows.Scan(&id, &prevHash, &currHash, &content, &fromUserID, &createdAt)

			// Chain break: prev_hash doesn't match previous record's curr_hash
			if prevHash != expectedPrev {
				alerts = append(alerts, gin.H{
					"target_type": "message", "target_id": id,
					"reason":   "哈希链断裂：prev_hash 与上一条记录不符（可能被删除或插入）",
					"expected": expectedPrev, "actual": prevHash,
				})
			}
			// Content tamper: recomputed hash != stored curr_hash
			recomputed := util.ComputeHash(prevHash, content, fromUserID, createdAt)
			if recomputed != currHash {
				alerts = append(alerts, gin.H{
					"target_type": "message", "target_id": id,
					"reason":   "内容被篡改：重新计算的哈希与存储值不符",
					"expected": recomputed, "actual": currHash,
				})
			}
			expectedPrev = currHash
		}
		rows.Close()
	}

	// 2. Check group messages chain
	rows, err = database.DB.Query(
		`SELECT id, prev_hash, curr_hash, content, from_user_id, created_at
		 FROM group_messages ORDER BY id ASC`)
	if err == nil {
		expectedPrev := ""
		for rows.Next() {
			var id, fromUserID int
			var prevHash, currHash, content, createdAt string
			rows.Scan(&id, &prevHash, &currHash, &content, &fromUserID, &createdAt)
			if prevHash != expectedPrev {
				alerts = append(alerts, gin.H{
					"target_type": "group_message", "target_id": id,
					"reason":   "群消息哈希链断裂：prev_hash 与上一条记录不符",
					"expected": expectedPrev, "actual": prevHash,
				})
			}
			recomputed := util.ComputeHash(prevHash, content, fromUserID, createdAt)
			if recomputed != currHash {
				alerts = append(alerts, gin.H{
					"target_type": "group_message", "target_id": id,
					"reason":   "群消息内容被篡改：重新计算的哈希与存储值不符",
					"expected": recomputed, "actual": currHash,
				})
			}
			expectedPrev = currHash
		}
		rows.Close()
	}

	// 3. Check audit log chain (the audit log protects itself)
	rows, err = database.DB.Query(
		`SELECT id, prev_hash, curr_hash, actor_id, action, detail, created_at
		 FROM audit_logs ORDER BY id ASC`)
	if err == nil {
		expectedPrev := ""
		for rows.Next() {
			var id, actorID int
			var prevHash, currHash, action, detail, createdAt string
			rows.Scan(&id, &prevHash, &currHash, &actorID, &action, &detail, &createdAt)
			if prevHash != expectedPrev {
				alerts = append(alerts, gin.H{
					"target_type": "audit_log", "target_id": id,
					"reason":   "审计日志链断裂：有人试图删除/插入审计记录",
					"expected": expectedPrev, "actual": prevHash,
				})
			}
			recomputed := util.ComputeAuditHash(prevHash, actorID, action, detail, createdAt)
			if recomputed != currHash {
				alerts = append(alerts, gin.H{
					"target_type": "audit_log", "target_id": id,
					"reason":   "审计日志被篡改：记录内容被修改",
					"expected": recomputed, "actual": currHash,
				})
			}
			expectedPrev = currHash
		}
		rows.Close()
	}

	// Record this check in the audit log
	userID := middleware.GetUserID(c)
	var uname string
	database.DB.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&uname)
	RecordAudit(userID, uname, "integrity_check",
		"发现"+strconv.Itoa(len(alerts))+"处异常", c.ClientIP())

	// If tampering found, persist alerts and notify admins.
	// Deduplicate: skip inserting when an UNHANDLED alert for the same
	// (target_type, target_id, reason) already exists, otherwise repeated
	// integrity checks would pile up duplicate rows (one per query) and the
	// admin panel would show the same tamper over and over. Once an admin marks
	// the alert as resolved (handled=1), a subsequent check will raise a fresh
	// alert if the tampering is still present.
	var newAlerts []gin.H
	if len(alerts) > 0 {
		for _, a := range alerts {
			var existingID int
			err := database.DB.QueryRow(
				`SELECT id FROM integrity_alerts
				 WHERE target_type = ? AND target_id = ? AND reason = ? AND handled = 0
				 ORDER BY id DESC LIMIT 1`,
				a["target_type"], a["target_id"], a["reason"],
			).Scan(&existingID)
			if err == nil {
				// An unhandled alert already exists for this record; skip to avoid duplicates.
				continue
			}
			database.DB.Exec(
				`INSERT INTO integrity_alerts (target_type, target_id, expected_hash, actual_hash, reason)
				 VALUES (?, ?, ?, ?, ?)`,
				a["target_type"], a["target_id"], a["expected"], a["actual"], a["reason"],
			)
			newAlerts = append(newAlerts, a)
		}
		// Push only newly-detected alerts to admins in real time
		if len(newAlerts) > 0 {
			pushIntegrityAlertToAdmins(newAlerts)
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Code: 200,
		Data: gin.H{
			"total_messages":   countTable("messages"),
			"total_group_msgs": countTable("group_messages"),
			"total_audit_logs": countTable("audit_logs"),
			"alerts":           alerts,
			"alert_count":      len(alerts),
			"status":           ternary(len(alerts) == 0, "完整 ✅", "已发现篡改 🚨"),
		},
	})
}

func countTable(name string) int {
	var n int
	database.DB.QueryRow("SELECT COUNT(*) FROM " + name).Scan(&n)
	return n
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func pushIntegrityAlertToAdmins(alerts []gin.H) {
	rows, err := database.DB.Query("SELECT id FROM users WHERE role = 1")
	if err != nil {
		return
	}
	data, _ := json.Marshal(alerts)
	for rows.Next() {
		var adminID int
		rows.Scan(&adminID)
		hub.DefaultHub.PushEvent(adminID, "integrity_alert", string(data))
	}
	rows.Close()
}

// TamperMessage simulates a direct database attack: modifies message content
// WITHOUT updating the hash chain. This is the attack demo (4th attack type).
func TamperMessage(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Code: 400, Message: "请提供新的content"})
		return
	}

	// VULNERABLE: directly modify content, do NOT touch curr_hash
	// In a real attack, a DBA or SQL-injection-with-write would do this.
	res, err := database.DB.Exec("UPDATE messages SET content = ? WHERE id = ?", req.Content, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "篡改失败"})
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, models.APIResponse{Code: 404, Message: "消息不存在"})
		return
	}

	// Record the attack in audit log (attacker's action is logged)
	userID := middleware.GetUserID(c)
	var uname string
	database.DB.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&uname)
	RecordAudit(userID, uname, "tamper",
		"篡改私聊消息#"+strconv.Itoa(id)+" 内容为:"+req.Content, c.ClientIP())

	c.JSON(http.StatusOK, models.APIResponse{
		Code:    200,
		Message: "已模拟直接篡改数据库（未更新哈希链）。现在调用 /api/integrity/check 即可检测到篡改。",
		Data: gin.H{
			"message_id":  id,
			"new_content": req.Content,
			"note":        "curr_hash 未更新，完整性校验将发现异常",
		},
	})
}

// GetAuditLogs returns audit log entries (admin only)
func GetAuditLogs(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, actor_id, actor_name, action, detail, ip_address, curr_hash, created_at
		 FROM audit_logs ORDER BY id DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var logs []gin.H
	for rows.Next() {
		var id, actorID int
		var actorName, action, detail, ip, hash, createdAt string
		rows.Scan(&id, &actorID, &actorName, &action, &detail, &ip, &hash, &createdAt)
		logs = append(logs, gin.H{
			"id": id, "actor_id": actorID, "actor_name": actorName,
			"action": action, "detail": detail, "ip": ip,
			"hash": hash, "created_at": createdAt,
		})
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: logs})
}

// GetIntegrityAlerts returns integrity alerts (admin only)
func GetIntegrityAlerts(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, target_type, target_id, expected_hash, actual_hash, reason, handled, created_at
		 FROM integrity_alerts ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "查询失败"})
		return
	}
	defer rows.Close()

	var alerts []gin.H
	for rows.Next() {
		var id, targetID, handled int
		var targetType, expected, actual, reason, createdAt string
		rows.Scan(&id, &targetType, &targetID, &expected, &actual, &reason, &handled, &createdAt)
		alerts = append(alerts, gin.H{
			"id": id, "target_type": targetType, "target_id": targetID,
			"expected_hash": expected, "actual_hash": actual, "reason": reason,
			"handled": handled, "created_at": createdAt,
		})
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Data: alerts})
}

// ResolveIntegrityAlert marks an integrity alert as handled (admin only)
func ResolveIntegrityAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	_, err := database.DB.Exec("UPDATE integrity_alerts SET handled = 1 WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Code: 500, Message: "更新失败"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Code: 200, Message: "已处理"})
}
