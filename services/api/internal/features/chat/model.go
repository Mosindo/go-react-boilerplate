package chat

import "time"

const (
	MaxMessageLength = 2000
	DefaultLimit     = 30
	MaxLimit         = 100
)

type Message struct {
	ID              string     `json:"id"`
	MatchID         string     `json:"matchId"`
	SenderID        string     `json:"senderId"`
	Body            string     `json:"body"`
	ClientMessageID *string    `json:"clientMessageId"`
	CreatedAt       time.Time  `json:"createdAt"`
	ReadAt          *time.Time `json:"readAt"`
}

type SendRequest struct {
	Body            string `json:"body" binding:"required,max=8000"`
	ClientMessageID string `json:"clientMessageId" binding:"omitempty,uuid"`
}

type ListResponse struct {
	Messages []Message `json:"messages"`
	// NextBefore is the cursor for older messages (null when exhausted).
	NextBefore *string `json:"nextBefore"`
}

type ReadEvent struct {
	MatchID  string    `json:"matchId"`
	ReaderID string    `json:"readerId"`
	ReadAt   time.Time `json:"readAt"`
}
