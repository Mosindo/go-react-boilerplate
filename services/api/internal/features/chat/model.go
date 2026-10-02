package chat

import "time"

const (
	MaxBodyLen = 2000
)

type Message struct {
	ID        int64      `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}
