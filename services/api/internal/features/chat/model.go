package chat

import (
	"errors"
	"time"
)

const (
	MaxMessageLength = 2000
	DefaultPageSize  = 30
	MaxPageSize      = 100
)

var (
	ErrNotFound     = errors.New("conversation not found")
	ErrEmptyMessage = errors.New("message cannot be empty")
	ErrTooLong      = errors.New("message is too long (max 2000 characters)")
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type Peer struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	PhotoURL  string `json:"photoUrl,omitempty"`
}

type Conversation struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"matchId"`
	Peer        Peer      `json:"peer"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
	CreatedAt   time.Time `json:"createdAt"`
	SortAt      time.Time `json:"sortAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}
