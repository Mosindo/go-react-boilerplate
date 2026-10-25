package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000
	DefaultPageSize = 30
)

var (
	ErrNotFound       = errors.New("conversation not found")
	ErrBlocked        = errors.New("conversation not available")
	ErrInvalidMessage = errors.New("message must be 1 to 2000 characters")
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type ConversationItem struct {
	ID          string                  `json:"id"`
	MatchID     string                  `json:"matchId"`
	OtherUser   profiles.ProfileSummary `json:"otherUser"`
	LastMessage *Message                `json:"lastMessage"`
	UnreadCount int                     `json:"unreadCount"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

type conversationRow struct {
	ID          string
	MatchID     string
	OtherUserID string
	UpdatedAt   time.Time
	UnreadCount int
	Last        Message
}

type readResult struct {
	OtherUserID string
	ReadAt      time.Time
	Count       int64
}

type sendResult struct {
	Message        Message
	OtherUserID    string
	NotificationID string
}
