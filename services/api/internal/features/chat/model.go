package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes     = 2000
	DefaultMessageLimit = 30
	MaxMessageLimit     = 100
	DefaultListLimit    = 30
	MaxListLimit        = 50
)

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	// Read is true once the other participant has read the message.
	Read bool `json:"read"`
}

type MessagePreview struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type StoredConversation struct {
	ID            string
	MatchID       string
	OtherUserID   string
	LastMessageAt time.Time
	UnreadCount   int
	LastMessage   MessagePreview
}

type Conversation struct {
	ID          string                 `json:"id"`
	MatchID     string                 `json:"matchId"`
	Profile     profiles.PublicProfile `json:"profile"`
	LastMessage MessagePreview         `json:"lastMessage"`
	UnreadCount int                    `json:"unreadCount"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextBefore    *time.Time     `json:"nextBefore"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	// NextBefore is the id to pass as `before` to load older messages, or null.
	NextBefore *int64 `json:"nextBefore"`
}

type SendRequest struct {
	Body string `json:"body"`
}

type ReadResult struct {
	ReadAt time.Time `json:"readAt"`
}
