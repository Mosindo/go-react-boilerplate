package notifications

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string                `json:"id"`
	Type      string                `json:"type"`
	Actor     *profiles.UserSummary `json:"actor"`
	Data      map[string]string     `json:"data"`
	ReadAt    *time.Time            `json:"readAt"`
	CreatedAt time.Time             `json:"createdAt"`
}

type ListResult struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    *string        `json:"nextCursor"`
}
