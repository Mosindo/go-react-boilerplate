package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string        `json:"id"`
	MatchID     string        `json:"matchId"`
	User        profiles.Card `json:"user"`
	LastMessage *Message      `json:"lastMessage"`
	UnreadCount int           `json:"unreadCount"`
	ActivityAt  time.Time     `json:"activityAt"`
}

type MessagesPage struct {
	Messages               []Message `json:"messages"`
	OtherLastReadMessageID int64     `json:"otherLastReadMessageId"`
}

type SendRequest struct {
	Body string `json:"body"`
}
