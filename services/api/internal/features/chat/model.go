package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes      = 2000
	DefaultConversations = 20
	MaxConversations     = 50
	DefaultMessagesLimit = 50
	MaxMessagesLimit     = 100
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
}

type Peer struct {
	UserID    string             `json:"userId"`
	FirstName string             `json:"firstName"`
	Age       int                `json:"age"`
	Photo     *profiles.PhotoRef `json:"photo,omitempty"`
}

type Conversation struct {
	ID            string    `json:"id"`
	MatchID       string    `json:"matchId"`
	Peer          Peer      `json:"peer"`
	LastMessage   *Message  `json:"lastMessage,omitempty"`
	UnreadCount   int       `json:"unreadCount"`
	LastMessageAt time.Time `json:"lastMessageAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *time.Time     `json:"nextCursor,omitempty"`
}

type MessagesResponse struct {
	// Messages are in chronological order (oldest first).
	Messages   []Message  `json:"messages"`
	NextCursor *time.Time `json:"nextCursor,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}
