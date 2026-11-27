package notifications

import "time"

// Kinds of notifications produced by the platform.
const (
	KindMatch = "match"
)

type Notification struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      map[string]string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    *time.Time        `json:"readAt,omitempty"`
}

type NotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	NextCursor    string                 `json:"nextCursor,omitempty"`
}

type UnreadCountResponse struct {
	Unread int `json:"unread"`
}
