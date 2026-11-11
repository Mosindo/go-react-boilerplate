package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type Message struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"matchId"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type conversationRow struct {
	MatchID         string
	OtherUserID     string
	CreatedAt       time.Time
	ActivityAt      time.Time
	LastMessage     *Message
	UnreadCount     int
	OtherLastReadAt *time.Time
}

type Conversation struct {
	MatchID         string        `json:"matchId"`
	CreatedAt       time.Time     `json:"createdAt"`
	User            profiles.Card `json:"user"`
	LastMessage     *Message      `json:"lastMessage"`
	UnreadCount     int           `json:"unreadCount"`
	OtherLastReadAt *time.Time    `json:"otherLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    string         `json:"nextCursor"`
}

type MessagesResponse struct {
	// Chronological order (oldest first) within the page.
	Messages        []Message  `json:"messages"`
	NextCursor      string     `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type ReadEvent struct {
	MatchID string    `json:"matchId"`
	UserID  string    `json:"userId"`
	ReadAt  time.Time `json:"readAt"`
}

type MessageEvent struct {
	Message Message `json:"message"`
}
