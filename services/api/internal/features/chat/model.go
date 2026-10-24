package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes     = 2000
	DefaultConvLimit    = 30
	MaxConvLimit        = 50
	DefaultMessageLimit = 40
	MaxMessageLimit     = 100
)

var (
	ErrNotFound       = errors.New("conversation not found")
	ErrEmptyMessage   = errors.New("message content required")
	ErrMessageTooLong = errors.New("message too long")
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

type Conversation struct {
	ID          string        `json:"id"`
	MatchID     string        `json:"matchId"`
	User        profiles.Card `json:"user"`
	LastMessage *Message      `json:"lastMessage,omitempty"`
	UnreadCount int           `json:"unreadCount"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type convRow struct {
	ID          string
	MatchID     string
	OtherUserID string
	Last        *Message
	UnreadCount int
	UpdatedAt   time.Time
}

// access describes what the caller may do in a conversation.
type access struct {
	OtherUserID string
	ClearedAt   *time.Time
}
