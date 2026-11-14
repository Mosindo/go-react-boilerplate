package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	maxMessageRunes = 2000
	maxUnreadCount  = 99
)

type Message struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"matchId"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	MatchID         string                 `json:"matchId"`
	User            profiles.PublicProfile `json:"user"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastMessage     *Message               `json:"lastMessage"`
	UnreadCount     int                    `json:"unreadCount"`
	OtherLastReadAt *time.Time             `json:"otherLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	HasMore         bool       `json:"hasMore"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type ReadResponse struct {
	ReadAt time.Time `json:"readAt"`
}

type conversationRow struct {
	MatchID         string
	OtherUserID     string
	CreatedAt       time.Time
	LastMessage     *Message
	UnreadCount     int
	OtherLastReadAt *time.Time
}

type membership struct {
	OtherUserID     string
	ClearedAt       *time.Time
	OtherLastReadAt *time.Time
}
