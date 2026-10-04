package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const MaxMessageRunes = 2000

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type ProfileCard struct {
	UserID      string          `json:"userId"`
	DisplayName string          `json:"displayName"`
	Age         int             `json:"age"`
	Photo       *profiles.Photo `json:"photo"`
}

type MatchSummary struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversationId"`
	MatchedAt      time.Time   `json:"matchedAt"`
	User           ProfileCard `json:"user"`
	LastMessage    *Message    `json:"lastMessage"`
	UnreadCount    int         `json:"unreadCount"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

type SendRequest struct {
	Body string `json:"body"`
}
