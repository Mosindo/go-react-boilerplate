package notifications

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Actor struct {
	UserID    string             `json:"userId"`
	FirstName string             `json:"firstName"`
	Photo     *profiles.PhotoRef `json:"photo,omitempty"`
}

type Notification struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Actor          *Actor     `json:"actor,omitempty"`
	ConversationID string     `json:"conversationId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    *time.Time     `json:"nextCursor,omitempty"`
}
