package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000
	DefaultPage     = 30
	MaxPage         = 100
)

var (
	ErrNotFound     = errors.New("conversation not found")
	ErrEmptyMessage = errors.New("message is empty")
	ErrTooLong      = errors.New("message too long")
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type Conversation struct {
	ID            string        `json:"id"`
	MatchID       string        `json:"matchId"`
	User          profiles.Card `json:"user"`
	LastMessage   *Message      `json:"lastMessage"`
	UnreadCount   int           `json:"unreadCount"`
	LastMessageAt time.Time     `json:"lastMessageAt"`
}

type conversationRow struct {
	ID, MatchID, OtherID string
	LastMessageAt        time.Time
	Last                 *Message
	Unread               int
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}
