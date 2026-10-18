package chat

import "time"

const (
	MaxMessageRunes = 2000
	DefaultPageSize = 30
	MaxPageSize     = 50
)

type UserSummary struct {
	UserID   string  `json:"userId"`
	Name     string  `json:"firstName"`
	ThumbURL *string `json:"thumbUrl"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationSummary struct {
	ID          string      `json:"id"`
	MatchID     string      `json:"matchId"`
	User        UserSummary `json:"user"`
	LastMessage *Message    `json:"lastMessage"`
	UnreadCount int         `json:"unreadCount"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type ConversationsResponse struct {
	Conversations []ConversationSummary `json:"conversations"`
	UnreadTotal   int                   `json:"unreadTotal"`
	NextCursor    string                `json:"nextCursor,omitempty"`
}

type MessagesResponse struct {
	// Messages are ordered newest first; pass NextCursor as ?cursor= for older ones.
	Messages        []Message  `json:"messages"`
	NextCursor      string     `json:"nextCursor,omitempty"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ReadResponse struct {
	ReadAt time.Time `json:"readAt"`
}
