package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const MaxMessageLength = 2000

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	// OtherLastReadAt lets the sender derive read receipts: a message is read when createdAt <= otherLastReadAt.
	OtherLastReadAt time.Time `json:"otherLastReadAt"`
	NextCursor      string    `json:"nextCursor,omitempty"`
}

type ConversationItem struct {
	ID            string                 `json:"id"`
	MatchID       string                 `json:"matchId"`
	User          profiles.PublicProfile `json:"user"`
	LastMessage   *Message               `json:"lastMessage"`
	LastMessageAt time.Time              `json:"lastMessageAt"`
	UnreadCount   int                    `json:"unreadCount"`
}

type ConversationsResponse struct {
	Conversations []ConversationItem `json:"conversations"`
	NextCursor    string             `json:"nextCursor,omitempty"`
}

type UnreadResponse struct {
	UnreadMessages      int `json:"unreadMessages"`
	UnreadConversations int `json:"unreadConversations"`
}
