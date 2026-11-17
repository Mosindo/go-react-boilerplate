package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultConversationsLimit = 20
	MaxConversationsLimit     = 50
	DefaultMessagesLimit      = 30
	MaxMessagesLimit          = 100
	MaxMessageRunes           = 2000
)

type Message struct {
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId"`
	SenderID        string    `json:"senderId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
	ClientMessageID string    `json:"clientMessageId,omitempty"`
}

type Access struct {
	ConversationID  string
	MatchID         string
	MatchedAt       time.Time
	OtherUserID     string
	HiddenAt        *time.Time
	OtherLastReadAt *time.Time
}

type StoredConversation struct {
	ID              string
	MatchID         string
	MatchedAt       time.Time
	OtherUserID     string
	OtherLastReadAt *time.Time
	LastMessage     *Message
	UnreadCount     int
}

type Conversation struct {
	ID              string                 `json:"id"`
	MatchID         string                 `json:"matchId"`
	MatchedAt       time.Time              `json:"matchedAt"`
	Profile         profiles.PublicProfile `json:"profile"`
	LastMessage     *Message               `json:"lastMessage"`
	UnreadCount     int                    `json:"unreadCount"`
	OtherLastReadAt *time.Time             `json:"otherLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesResponse struct {
	// Messages are returned newest first; clients reverse them for display.
	Messages        []Message  `json:"messages"`
	NextCursor      *string    `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Content         string `json:"content" binding:"required,max=8000"`
	ClientMessageID string `json:"clientMessageId" binding:"max=64"`
}

type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	UserID         string    `json:"userId"`
	ReadAt         time.Time `json:"readAt"`
}
