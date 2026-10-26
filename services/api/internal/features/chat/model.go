package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxBodyLength       = 2000
	DefaultMessagePage  = 30
	MaxMessagePage      = 100
	DefaultConversation = 30
	MaxConversation     = 50
)

type Message struct {
	ID        int64      `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type Conversation struct {
	MatchID     string                 `json:"matchId"`
	MatchedAt   time.Time              `json:"matchedAt"`
	Profile     profiles.PublicProfile `json:"profile"`
	LastMessage *Message               `json:"lastMessage"`
	UnreadCount int                    `json:"unreadCount"`
	ActivityAt  time.Time              `json:"activityAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}

type MessagesResponse struct {
	// Messages are ordered newest first; pass the smallest id as "before" to page back.
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ReadReceipt struct {
	MatchID  string    `json:"matchId"`
	ReaderID string    `json:"readerId"`
	ReadAt   time.Time `json:"readAt"`
	Count    int       `json:"count"`
}
