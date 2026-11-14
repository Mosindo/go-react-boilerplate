package notifications

import (
	"encoding/json"
	"time"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
	TypeSystem  = "system"
)

type Notification struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	IsRead    bool            `json:"isRead"`
	CreatedAt time.Time       `json:"createdAt"`
	ReadAt    *time.Time      `json:"readAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
}
