package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageLength = 2000
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationItem struct {
	ID          string               `json:"id"`
	User        profiles.UserSummary `json:"user"`
	LastMessage *Message             `json:"lastMessage"`
	UnreadCount int                  `json:"unreadCount"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

type ConversationsResult struct {
	Conversations []ConversationItem `json:"conversations"`
	NextCursor    *string            `json:"nextCursor"`
}

type MessagesResult struct {
	Messages      []Message  `json:"messages"`
	OtherLastRead *time.Time `json:"otherLastReadAt"`
	NextCursor    *string    `json:"nextCursor"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

// participant describes the caller's membership of a conversation.
type participant struct {
	OtherUserID string
	HiddenAt    *time.Time
}
