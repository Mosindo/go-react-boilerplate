package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000
)

type Message struct {
	ID        string     `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

type StoredConversation struct {
	MatchID     string
	OtherID     string
	CreatedAt   time.Time
	ActivityAt  time.Time
	LastMessage *Message
	Unread      int
}

type ConversationResponse struct {
	MatchID     string                 `json:"matchId"`
	CreatedAt   time.Time              `json:"createdAt"`
	ActivityAt  time.Time              `json:"activityAt"`
	Profile     profiles.PublicProfile `json:"profile"`
	LastMessage *Message               `json:"lastMessage"`
	UnreadCount int                    `json:"unreadCount"`
}

type MessagesPage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}
