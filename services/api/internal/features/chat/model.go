package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000

	defaultConversationsLimit = 30
	maxConversationsLimit     = 100
	defaultMessagesLimit      = 40
	maxMessagesLimit          = 100
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderUserID   string     `json:"senderUserId"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

type ConversationPreview struct {
	ID          string        `json:"id"`
	MatchID     string        `json:"matchId"`
	User        profiles.Card `json:"user"`
	LastMessage *Message      `json:"lastMessage,omitempty"`
	UnreadCount int           `json:"unreadCount"`
}

type ConversationsResponse struct {
	Conversations []ConversationPreview `json:"conversations"`
}

type MessagesResponse struct {
	// Messages are in chronological order (oldest first).
	Messages []Message `json:"messages"`
	// NextCursor is the id to pass as ?before= to load older messages; empty when the history is exhausted.
	NextCursor string `json:"nextCursor,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ReadResponse struct {
	UnreadCount int `json:"unreadCount"`
}
