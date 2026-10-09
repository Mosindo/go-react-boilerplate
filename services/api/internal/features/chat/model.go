package chat

import (
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000
	DefaultPageSize = 30
	MaxPageSize     = 50
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type LastMessage struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string                  `json:"id"`
	MatchID     string                  `json:"matchId"`
	User        profiles.ProfileSummary `json:"user"`
	LastMessage *LastMessage            `json:"lastMessage"`
	UnreadCount int                     `json:"unreadCount"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

// SentMessage is the outcome of sending: what to push and to whom.
type SentMessage struct {
	Message      Message
	RecipientID  string
	Notification notifications.Notification
}

// ReadReceipt is the outcome of marking a conversation as read.
type ReadReceipt struct {
	OtherUserID string
	ReadAt      time.Time
}
