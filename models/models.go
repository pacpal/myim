package models

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Role      int       `json:"role"`
	Status    int       `json:"status"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
}

type Friend struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	FriendID  int       `json:"friend_id"`
	Remark    string    `json:"remark"`
	Nickname  string    `json:"nickname"`
	Username  string    `json:"username"`
	Avatar    string    `json:"avatar"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendRequest struct {
	ID           int       `json:"id"`
	FromUserID   int       `json:"from_user_id"`
	ToUserID     int       `json:"to_user_id"`
	FromNickname string    `json:"from_nickname"`
	FromUsername string    `json:"from_username"`
	Message      string    `json:"message"`
	Status       int       `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type Group struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	OwnerID     int       `json:"owner_id"`
	OwnerName   string    `json:"owner_name"`
	Avatar      string    `json:"avatar"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type GroupMember struct {
	ID       int       `json:"id"`
	GroupID  int       `json:"group_id"`
	UserID   int       `json:"user_id"`
	Username string    `json:"username"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Role     int       `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type Message struct {
	ID         int       `json:"id"`
	FromUserID int       `json:"from_user_id"`
	ToUserID   int       `json:"to_user_id"`
	FromName   string    `json:"from_name"`
	ToName     string    `json:"to_name"`
	Content    string    `json:"content"`
	MsgType    int       `json:"msg_type"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type GroupMessage struct {
	ID         int       `json:"id"`
	GroupID    int       `json:"group_id"`
	FromUserID int       `json:"from_user_id"`
	FromName   string    `json:"from_name"`
	Content    string    `json:"content"`
	MsgType    int       `json:"msg_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type LoginLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
