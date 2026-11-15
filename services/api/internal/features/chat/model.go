package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const MaxMessageRunes = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationRow struct {
	ID              string
	MatchID         string
	MatchedAt       time.Time
	OtherUserID     string
	OtherLastReadAt *time.Time
	LastMessage     *Message
	UnreadCount     int
	SortAt          time.Time
}

type ConversationResponse struct {
	ID              string        `json:"id"`
	MatchID         string        `json:"matchId"`
	MatchedAt       time.Time     `json:"matchedAt"`
	User            profiles.Card `json:"user"`
	LastMessage     *Message      `json:"lastMessage"`
	UnreadCount     int           `json:"unreadCount"`
	OtherLastReadAt *time.Time    `json:"otherLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	NextCursor    *string                `json:"nextCursor"`
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	NextCursor      *string    `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

// Participants of a conversation, as seen by one member.
type Membership struct {
	ConversationID  string
	OtherUserID     string
	ClearedAt       *time.Time
	OtherLastReadAt *time.Time
}
