package notifications

import "time"

const (
	TypeMatch   = "match.new"
	TypeMessage = "message.new"
)

type Notification struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    *time.Time        `json:"readAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
	UnreadCount   int            `json:"unreadCount"`
}
