package chat

import (
	"time"

	"example.com/api/internal/features/photos"
)

const MaxMessageRunes = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Peer struct {
	UserID    string        `json:"userId"`
	FirstName string        `json:"firstName"`
	Photo     *photos.Photo `json:"photo"`
}

type Conversation struct {
	ID             string     `json:"id"`
	MatchID        string     `json:"matchId"`
	MatchedAt      time.Time  `json:"matchedAt"`
	Peer           Peer       `json:"peer"`
	LastMessage    *Message   `json:"lastMessage"`
	UnreadCount    int        `json:"unreadCount"`
	PeerLastReadAt *time.Time `json:"peerLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	HasMore       bool           `json:"hasMore"`
}

type MessagesResponse struct {
	Messages       []Message  `json:"messages"`
	NextCursor     *string    `json:"nextCursor"`
	PeerLastReadAt *time.Time `json:"peerLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type UnreadResponse struct {
	UnreadCount int `json:"unreadCount"`
}

type membership struct {
	PeerID         string
	ClearedAt      *time.Time
	PeerLastReadAt *time.Time
}
