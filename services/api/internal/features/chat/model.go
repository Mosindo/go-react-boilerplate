package chat

import "time"

const (
	DefaultLimit = 30
	MaxLimit     = 50
	MaxBodyLen   = 2000
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type SendRequest struct {
	Body string `json:"body"`
}

type Cursor struct {
	CreatedAt time.Time `json:"t"`
	ID        string    `json:"id"`
}

type ReadEvent struct {
	ConversationID string `json:"conversationId"`
	ReaderID       string `json:"readerId"`
}
