package matches

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes      = 2000
	DefaultMatchesLimit  = 20
	MaxMatchesLimit      = 50
	DefaultMessagesLimit = 30
	MaxMessagesLimit     = 100
)

type Message struct {
	ID        string
	MatchID   string
	SenderID  string
	Body      string
	CreatedAt time.Time
	ReadAt    *time.Time
}

type MatchSummary struct {
	ID             string
	OtherUserID    string
	CreatedAt      time.Time
	LastActivityAt time.Time
	LastMessage    *Message
	UnreadCount    int
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type MessageResponse struct {
	ID        string     `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type MatchResponse struct {
	ID             string                `json:"id"`
	CreatedAt      time.Time             `json:"createdAt"`
	LastActivityAt time.Time             `json:"lastActivityAt"`
	Profile        profiles.CardResponse `json:"profile"`
	LastMessage    *MessageResponse      `json:"lastMessage"`
	UnreadCount    int                   `json:"unreadCount"`
}

type MatchesResponse struct {
	Matches    []MatchResponse `json:"matches"`
	NextCursor *string         `json:"nextCursor"`
}

type MessagesResponse struct {
	Messages   []MessageResponse `json:"messages"`
	NextCursor *string           `json:"nextCursor"`
}

func ToMessageResponse(m Message) MessageResponse {
	return MessageResponse{ID: m.ID, MatchID: m.MatchID, SenderID: m.SenderID, Body: m.Body, CreatedAt: m.CreatedAt, ReadAt: m.ReadAt}
}
