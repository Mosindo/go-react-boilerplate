package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
	ReadAt    *time.Time      `json:"readAt,omitempty"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextOffset    *int           `json:"nextOffset,omitempty"`
}

type PushTokenRequest struct {
	Token    string `json:"token" binding:"required,max=200"`
	Platform string `json:"platform" binding:"required,oneof=ios android"`
}

type DeletePushTokenRequest struct {
	Token string `json:"token" binding:"required,max=200"`
}
