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
	ID             string        `json:"id"`
	Peer           profiles.Card `json:"peer"`
	LastMessage    *Message      `json:"lastMessage,omitempty"`
	UnreadCount    int           `json:"unreadCount"`
	PeerLastReadAt time.Time     `json:"peerLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextBefore    *time.Time     `json:"nextBefore,omitempty"`
}

type MessagesResponse struct {
	Messages       []Message  `json:"messages"` // newest first
	NextBefore     *time.Time `json:"nextBefore,omitempty"`
	PeerLastReadAt time.Time  `json:"peerLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type storedConversation struct {
	ID             string
	PeerID         string
	LastMessage    *Message
	UnreadCount    int
	PeerLastReadAt time.Time
	SortAt         time.Time
}

type participantInfo struct {
	PeerID         string
	PeerLastReadAt time.Time
	Blocked        bool
}
