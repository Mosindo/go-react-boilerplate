package chat

import (
	"time"

	"example.com/api/internal/platform/storage"
)

const maxMessageRunes = 2000

type SendRequest struct {
	Body string `json:"body"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string              `json:"id"`
	MatchID     string              `json:"matchId"`
	User        storage.UserSummary `json:"user"`
	LastMessage *LastMessage        `json:"lastMessage"`
	UnreadCount int                 `json:"unreadCount"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type ConversationPage struct {
	Items      []Conversation `json:"items"`
	NextBefore *string        `json:"nextBefore"`
}

// MessagePage lists messages newest first; OtherLastReadAt drives read receipts on the caller's messages.
type MessagePage struct {
	Items           []Message  `json:"items"`
	NextBefore      *int64     `json:"nextBefore"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}
