package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	ReadAt    *time.Time        `json:"readAt"`
	CreatedAt time.Time         `json:"createdAt"`
}

// NewNotification describes a notification to create. A non-empty DedupeKey
// coalesces it with an existing *unread* notification of the same user and key
// (used so a chat with 50 new messages shows one notification, not 50).
type NewNotification struct {
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      map[string]string
	DedupeKey string
}

type NotificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
}
