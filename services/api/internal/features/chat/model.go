package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID             string               `json:"id"`
	MatchID        string               `json:"matchId"`
	User           profiles.UserSummary `json:"user"`
	LastMessage    *Message             `json:"lastMessage"`
	UnreadCount    int                  `json:"unreadCount"`
	PeerLastReadAt *time.Time           `json:"peerLastReadAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

type MessagesPage struct {
	Items      []Message `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	ReaderID       string    `json:"readerId"`
	ReadAt         time.Time `json:"readAt"`
}
