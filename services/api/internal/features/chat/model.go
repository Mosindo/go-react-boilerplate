package chat

import "time"

const (
	MaxMessageLength = 2000
	DefaultPageSize  = 50
	MaxPageSize      = 100
)

type Message struct {
	ID             int64      `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type ConversationUser struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Conversation struct {
	ID          string           `json:"id"`
	MatchID     string           `json:"matchId"`
	MatchedAt   time.Time        `json:"matchedAt"`
	User        ConversationUser `json:"user"`
	LastMessage *Message         `json:"lastMessage"`
	UnreadCount int              `json:"unreadCount"`
	// SortAt is the activity time conversations are ordered by (and the paging cursor).
	SortAt time.Time `json:"sortAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=10000"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextBefore    *time.Time     `json:"nextBefore"`
}

type MessagesResponse struct {
	Messages   []Message `json:"messages"`
	NextBefore *int64    `json:"nextBefore"`
}

type ReadResponse struct {
	ReadCount int `json:"readCount"`
}
