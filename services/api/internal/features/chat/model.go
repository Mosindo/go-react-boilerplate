package chat

import (
	"time"

	"example.com/api/internal/features/photos"
)

const (
	DefaultConversationsLimit = 20
	MaxConversationsLimit     = 50
	DefaultMessagesLimit      = 30
	MaxMessagesLimit          = 100
	MaxMessageRunes           = 2000
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	IsRead         bool      `json:"isRead"`
}

type Participant struct {
	UserID    string                `json:"userId"`
	FirstName string                `json:"firstName"`
	Age       int                   `json:"age"`
	Photo     *photos.PhotoResponse `json:"photo"`
}

type Conversation struct {
	ID            string      `json:"id"`
	MatchID       string      `json:"matchId"`
	MatchedAt     time.Time   `json:"matchedAt"`
	LastMessage   *Message    `json:"lastMessage"`
	UnreadCount   int         `json:"unreadCount"`
	OtherUser     Participant `json:"otherUser"`
	otherBirthday time.Time
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	NextCursor      *string    `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

// Membership describes the caller's seat in a conversation.
type Membership struct {
	ConversationID  string
	OtherUserID     string
	HiddenAt        *time.Time
	OtherLastReadAt *time.Time
	MyLastReadAt    *time.Time
}
