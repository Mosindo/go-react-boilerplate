package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const maxMessageRunes = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID              string                 `json:"id"`
	MatchID         string                 `json:"matchId"`
	Profile         profiles.PublicProfile `json:"profile"`
	LastMessage     *Message               `json:"lastMessage"`
	UnreadCount     int                    `json:"unreadCount"`
	OtherLastReadAt *time.Time             `json:"otherLastReadAt"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type ConversationsPage struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesPage struct {
	// Messages are ordered newest first.
	Messages        []Message  `json:"messages"`
	NextCursor      *string    `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

type ReadReceipt struct {
	ConversationID string    `json:"conversationId"`
	UserID         string    `json:"userId"`
	LastReadAt     time.Time `json:"lastReadAt"`
}

type membership struct {
	ConversationID  string
	MatchID         string
	OtherUserID     string
	OtherLastReadAt *time.Time
	CreatedAt       time.Time
}

type conversationRow struct {
	membership
	LastMessage    *Message
	UnreadCount    int
	LastActivityAt time.Time
}
