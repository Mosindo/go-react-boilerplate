package notifications

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type Notification struct {
	ID             string                   `json:"id"`
	Type           string                   `json:"type"`
	Actor          *profiles.ProfileSummary `json:"actor"`
	ConversationID *string                  `json:"conversationId"`
	ReadAt         *time.Time               `json:"readAt"`
	CreatedAt      time.Time                `json:"createdAt"`
}

type ListResponse struct {
	Items       []Notification `json:"items"`
	NextCursor  *string        `json:"nextCursor"`
	UnreadCount int            `json:"unreadCount"`
}

type row struct {
	ID             string
	Type           string
	ActorID        *string
	ConversationID *string
	ReadAt         *time.Time
	CreatedAt      time.Time
}
