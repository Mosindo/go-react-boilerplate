package notifications

import (
	"errors"
	"time"
)

const (
	TypeNewMatch   = "new_match"
	TypeNewMessage = "new_message"

	DefaultLimit = 20
	MaxLimit     = 50
)

var (
	ErrNotFound      = errors.New("notification not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Actor struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoID   *string `json:"photoId,omitempty"`
}

type Notification struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Actor          *Actor     `json:"actor,omitempty"`
	ConversationID *string    `json:"conversationId,omitempty"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}
