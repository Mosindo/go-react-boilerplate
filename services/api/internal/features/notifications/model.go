package notifications

import (
	"time"

	"example.com/api/internal/platform/storage"
)

type Notification struct {
	ID             string              `json:"id"`
	Type           string              `json:"type"`
	ConversationID *string             `json:"conversationId"`
	Actor          storage.UserSummary `json:"actor"`
	ReadAt         *time.Time          `json:"readAt"`
	CreatedAt      time.Time           `json:"createdAt"`
}

type Page struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
	NextBefore  *string        `json:"nextBefore"`
}

type MarkReadRequest struct {
	IDs []string `json:"ids"`
}
