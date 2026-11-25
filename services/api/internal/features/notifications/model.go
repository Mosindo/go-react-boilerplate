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
	ReadAt    *time.Time      `json:"readAt"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
	UnreadCount   int            `json:"unreadCount"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unreadCount"`
}
