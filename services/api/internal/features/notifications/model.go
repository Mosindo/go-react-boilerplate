package notifications

import "time"

// AppNotification is the API shape (see docs/API.md).
type AppNotification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data"`
	IsRead    bool           `json:"isRead"`
	CreatedAt time.Time      `json:"createdAt"`
}

type ListResult struct {
	Notifications []AppNotification `json:"notifications"`
	UnreadCount   int               `json:"unreadCount"`
}

const (
	defaultLimit = 30
	maxLimit     = 100
)
