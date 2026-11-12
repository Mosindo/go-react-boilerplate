package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const maxMessageRunes = 2000

var (
	ErrNotMatched           = errors.New("you can only chat with your matches")
	ErrMessageContentNeeded = errors.New("message content required")
	ErrMessageTooLong       = errors.New("message is too long (2000 characters max)")
	ErrInvalidCursor        = errors.New("invalid cursor")
)

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type Message struct {
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type MessagePreview struct {
	ID           string    `json:"id"`
	SenderUserID string    `json:"senderUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ChatSummary struct {
	MatchID        string                 `json:"matchId"`
	ConversationID string                 `json:"conversationId"`
	User           profiles.PublicProfile `json:"user"`
	LastMessage    *MessagePreview        `json:"lastMessage,omitempty"`
	UnreadCount    int                    `json:"unreadCount"`
	MatchedAt      time.Time              `json:"matchedAt"`
}

type ChatsResponse struct {
	Chats []ChatSummary `json:"chats"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	// OtherLastReadAt lets the client render "seen" on its own messages.
	OtherLastReadAt *time.Time `json:"otherLastReadAt,omitempty"`
	HasMore         bool       `json:"hasMore"`
}

type ReadResponse struct {
	ReadAt time.Time `json:"readAt"`
}

type UnreadResponse struct {
	UnreadConversations int `json:"unreadConversations"`
}

type conversationRef struct {
	ConversationID string
	MatchID        string
}

type storedSummary struct {
	MatchID        string
	ConversationID string
	OtherUserID    string
	LastMessage    *MessagePreview
	UnreadCount    int
	MatchedAt      time.Time
}
