package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const MaxMessageRunes = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string           `json:"id"`
	MatchID     string           `json:"matchId"`
	MatchedAt   time.Time        `json:"matchedAt"`
	User        profiles.Summary `json:"user"`
	LastMessage *Message         `json:"lastMessage"`
	UnreadCount int              `json:"unreadCount"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesResponse struct {
	// Messages are ordered newest first.
	Messages []Message `json:"messages"`
	// OtherLastReadAt drives read receipts: a sent message is "read" when
	// its createdAt is <= this value.
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	NextCursor      *string    `json:"nextCursor"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	ReaderID       string    `json:"readerId"`
	LastReadAt     time.Time `json:"lastReadAt"`
}

type conversationRow struct {
	ID          string
	MatchID     string
	MatchedAt   time.Time
	OtherUserID string
	LastMessage *Message
	UnreadCount int
	ActivityAt  time.Time
}

// cursor is a keyset position (timestamp + id tie-breaker).
type cursor struct {
	At time.Time
	ID string
}
