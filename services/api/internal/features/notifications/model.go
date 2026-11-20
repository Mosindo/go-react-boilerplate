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
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      json.RawMessage
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationResponse struct {
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
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int                    `json:"unreadCount"`
}

func toResponse(n Notification) NotificationResponse {
	data := n.Data
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	return NotificationResponse{ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Data: data, IsRead: n.IsRead, CreatedAt: n.CreatedAt, ReadAt: n.ReadAt}
}
