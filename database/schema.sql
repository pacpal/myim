-- Weak-Network Optimized IM System Database Schema
-- Database: im_system

CREATE DATABASE IF NOT EXISTS im_system DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE im_system;

-- 1. users table - user accounts
-- role: 0=normal user, 1=admin (audit center access)
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    nickname VARCHAR(50) NOT NULL,
    avatar VARCHAR(255) DEFAULT '',
    role TINYINT DEFAULT 0 COMMENT '0=normal, 1=admin',
    status TINYINT DEFAULT 0 COMMENT '0=offline, 1=online',
    last_login DATETIME DEFAULT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. friends table - friend relationships
CREATE TABLE IF NOT EXISTS friends (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    friend_id INT NOT NULL,
    remark VARCHAR(50) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (friend_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_friend (user_id, friend_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. friend_requests table - friend requests
CREATE TABLE IF NOT EXISTS friend_requests (
    id INT AUTO_INCREMENT PRIMARY KEY,
    from_user_id INT NOT NULL,
    to_user_id INT NOT NULL,
    message VARCHAR(255) DEFAULT '',
    status TINYINT DEFAULT 0 COMMENT '0=pending, 1=accepted, 2=rejected',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. groups table - group chats
CREATE TABLE IF NOT EXISTS `groups` (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    owner_id INT NOT NULL,
    avatar VARCHAR(255) DEFAULT '',
    description VARCHAR(255) DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. group_members table - group membership
CREATE TABLE IF NOT EXISTS group_members (
    id INT AUTO_INCREMENT PRIMARY KEY,
    group_id INT NOT NULL,
    user_id INT NOT NULL,
    role TINYINT DEFAULT 0 COMMENT '0=member, 1=admin, 2=owner',
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_group_user (group_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 6. messages table - private messages (with hash chain for tamper detection)
CREATE TABLE IF NOT EXISTS messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    from_user_id INT NOT NULL,
    to_user_id INT NOT NULL,
    content TEXT NOT NULL,
    msg_type TINYINT DEFAULT 0 COMMENT '0=text, 1=image, 2=file',
    status TINYINT DEFAULT 0 COMMENT '0=sent, 1=delivered, 2=read',
    prev_hash CHAR(64) DEFAULT '' COMMENT 'hash of previous message in chain',
    curr_hash CHAR(64) DEFAULT '' COMMENT 'SHA256(prev_hash||content||from_user_id||created_at)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_from_to (from_user_id, to_user_id),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 7. group_messages table - group messages
CREATE TABLE IF NOT EXISTS group_messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    group_id INT NOT NULL,
    from_user_id INT NOT NULL,
    content TEXT NOT NULL,
    msg_type TINYINT DEFAULT 0,
    prev_hash CHAR(64) DEFAULT '' COMMENT 'hash of previous group message',
    curr_hash CHAR(64) DEFAULT '' COMMENT 'SHA256(prev_hash||content||from_user_id||created_at)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE,
    FOREIGN KEY (from_user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_group_created (group_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 8. message_ack table - message acknowledgments (weak network reliability)
CREATE TABLE IF NOT EXISTS message_ack (
    id INT AUTO_INCREMENT PRIMARY KEY,
    message_id INT NOT NULL,
    user_id INT NOT NULL,
    ack_type TINYINT DEFAULT 0 COMMENT '0=received, 1=read',
    ack_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uk_msg_user (message_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 9. offline_messages table - offline message storage (weak network)
CREATE TABLE IF NOT EXISTS offline_messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    message_id INT NOT NULL,
    delivered TINYINT DEFAULT 0 COMMENT '0=not delivered, 1=delivered',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    INDEX idx_user_delivered (user_id, delivered)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 10. login_logs table - login logs (for security audit)
CREATE TABLE IF NOT EXISTS login_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT DEFAULT NULL,
    username VARCHAR(50) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent VARCHAR(255) DEFAULT '',
    status TINYINT DEFAULT 0 COMMENT '0=fail, 1=success',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_username_time (username, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Demo users will be created via the registration page.
-- The password hashes are generated at runtime using bcrypt.

-- ============================================================
-- Tamper-proof audit IM extension: hash chain + audit logs
-- ============================================================

-- 11. audit_logs table - tamper-proof audit log (self-protected by hash chain)
-- Records all sensitive operations; any modification to this table is detectable.
CREATE TABLE IF NOT EXISTS audit_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    actor_id INT DEFAULT NULL COMMENT 'user performing the action',
    actor_name VARCHAR(50) DEFAULT '',
    action VARCHAR(50) NOT NULL COMMENT 'login/logout/send_msg/tamper/integrity_check etc.',
    detail VARCHAR(500) DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    prev_hash CHAR(64) DEFAULT '' COMMENT 'hash of previous audit log entry',
    curr_hash CHAR(64) DEFAULT '' COMMENT 'SHA256(prev_hash||actor_id||action||detail||created_at)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_action (action),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 12. integrity_alerts table - alerts raised when tampering is detected
CREATE TABLE IF NOT EXISTS integrity_alerts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    target_type VARCHAR(20) NOT NULL COMMENT 'message/group_message/audit_log',
    target_id INT NOT NULL COMMENT 'id of tampered record',
    expected_hash CHAR(64) DEFAULT '',
    actual_hash CHAR(64) DEFAULT '',
    reason VARCHAR(255) DEFAULT '',
    detected_by INT DEFAULT NULL COMMENT 'admin user id, NULL if auto',
    handled TINYINT DEFAULT 0 COMMENT '0=new, 1=resolved',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_handled (handled),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;