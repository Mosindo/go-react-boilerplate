package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes     = 2000
	DefaultMessagesPage = 30
	MaxMessagesPage     = 100
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type ConversationSummary struct {
	ID          string                 `json:"id"`
	MatchID     string                 `json:"matchId"`
	User        profiles.PublicProfile `json:"user"`
	LastMessage Message                `json:"lastMessage"`
	UnreadCount int                    `json:"unreadCount"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ConversationsResponse struct {
	Conversations []ConversationSummary `json:"conversations"`
	TotalUnread   int                   `json:"totalUnread"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

type ReadResponse struct {
	Marked int `json:"marked"`
}

// convoRow is the repository shape of a conversation list entry.
type convoRow struct {
	ID          string
	MatchID     string
	OtherUserID string
	Last        Message
	Unread      int
}
