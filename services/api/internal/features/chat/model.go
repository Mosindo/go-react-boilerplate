package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000
	DefaultPage     = 30
	MaxPage         = 100
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MessagePreview struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string          `json:"id"`
	MatchID     string          `json:"matchId"`
	User        profiles.Card   `json:"user"`
	LastMessage *MessagePreview `json:"lastMessage,omitempty"`
	UnreadCount int             `json:"unreadCount"`
	CreatedAt   time.Time       `json:"createdAt"`
	SortAt      time.Time       `json:"sortAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type MessagesResponse struct {
	Messages   []Message  `json:"messages"`
	NextBefore *time.Time `json:"nextBefore,omitempty"`
	// OtherLastReadAt lets the client render "read" receipts.
	OtherLastReadAt *time.Time `json:"otherLastReadAt,omitempty"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

// storedConversation is a conversation row for one participant.
type storedConversation struct {
	ID            string
	MatchID       string
	OtherUserID   string
	CreatedAt     time.Time
	LastBody      *string
	LastSenderID  *string
	LastMessageAt *time.Time
	UnreadCount   int
}
