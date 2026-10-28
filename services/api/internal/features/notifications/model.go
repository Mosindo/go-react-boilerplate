package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Actor struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
}

type Notification struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	ConversationID string     `json:"conversationId"`
	Actor          Actor      `json:"actor"`
	ReadAt         *time.Time `json:"readAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Page struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"nextCursor"`
	UnreadCount   int            `json:"unreadCount"`
}
