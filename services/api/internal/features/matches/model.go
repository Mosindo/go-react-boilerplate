package matches

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type MatchUser struct {
	UserID    string          `json:"userId"`
	FirstName string          `json:"firstName"`
	Age       int             `json:"age"`
	Photo     *profiles.Photo `json:"photo"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Summary struct {
	MatchID        string       `json:"matchId"`
	ConversationID string       `json:"conversationId"`
	CreatedAt      time.Time    `json:"createdAt"`
	User           MatchUser    `json:"user"`
	LastMessage    *LastMessage `json:"lastMessage"`
	UnreadCount    int          `json:"unreadCount"`
}

// Row is a Summary plus the data the service turns into derived fields.
type Row struct {
	Summary
	BirthDate time.Time
	SortAt    time.Time
}

type Cursor struct {
	SortAt time.Time `json:"t"`
	ID     string    `json:"id"`
}

type RemovedEvent struct {
	MatchID        string `json:"matchId"`
	ConversationID string `json:"conversationId"`
}
