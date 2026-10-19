package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes = 2000

	ConversationsDefaultLimit = 20
	ConversationsMaxLimit     = 50
	MessagesDefaultLimit      = 30
	MessagesMaxLimit          = 100
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrInvalidMessage       = errors.New("invalid message")
	ErrInvalidCursor        = errors.New("invalid cursor")
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string                 `json:"id"`
	MatchID     string                 `json:"matchId"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Profile     profiles.PublicProfile `json:"profile"`
	LastMessage *Message               `json:"lastMessage,omitempty"`
	UnreadCount int                    `json:"unreadCount"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}

type MessagesResponse struct {
	// Messages are ordered newest first.
	Messages []Message `json:"messages"`
	// PeerLastReadAt lets the client render read receipts: a message of mine is
	// read when createdAt <= peerLastReadAt.
	PeerLastReadAt time.Time `json:"peerLastReadAt"`
	NextCursor     string    `json:"nextCursor,omitempty"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

// Sent is the repository result of storing a message.
type Sent struct {
	Message           Message
	PeerID            string
	NotificationAdded bool
}
