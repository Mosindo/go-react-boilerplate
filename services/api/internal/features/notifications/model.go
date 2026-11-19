package notifications

import (
	"errors"
	"time"
)

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

var ErrNotFound = errors.New("notification not found")

type Stored struct {
	ID             string
	Type           string
	ActorID        *string
	ConversationID *string
	ReadAt         *time.Time
	CreatedAt      time.Time
}

type Actor struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Notification struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Actor          *Actor     `json:"actor"`
	ConversationID *string    `json:"conversationId"`
	ReadAt         *time.Time `json:"readAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
	UnreadCount   int            `json:"unreadCount"`
}
