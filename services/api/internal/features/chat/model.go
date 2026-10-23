package chat

import "time"

const (
	DefaultLimit   = 30
	MaxLimit       = 100
	MaxContentRune = 2000
)

type Message struct {
	ID        string     `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type SendRequest struct {
	Content string `json:"content"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor *string   `json:"nextCursor"`
}

// Member describes the caller's side of a conversation.
type Member struct {
	OtherUserID string
	SelfName    string
}

type ReadReceipt struct {
	MatchID  string    `json:"matchId"`
	ReaderID string    `json:"readerId"`
	ReadAt   time.Time `json:"readAt"`
}
