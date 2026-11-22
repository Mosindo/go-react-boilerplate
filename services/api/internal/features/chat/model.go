package chat

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const maxMessageRunes = 2000

var (
	ErrNotMatched           = errors.New("you can only message your matches")
	ErrMessageContentNeeded = errors.New("message content required")
	ErrMessageTooLong       = errors.New("message is too long")
	ErrInvalidCursor        = errors.New("invalid cursor")
)

type ChatUser struct {
	ID        string          `json:"id"`
	FirstName string          `json:"firstName"`
	Photo     *profiles.Photo `json:"photo,omitempty"`
	IsDemo    bool            `json:"isDemo"`
}

type MessagePreview struct {
	ID           string    `json:"id"`
	SenderUserID string    `json:"senderUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ChatSummary struct {
	MatchID        string          `json:"matchId"`
	ConversationID string          `json:"conversationId"`
	MatchedAt      time.Time       `json:"matchedAt"`
	User           ChatUser        `json:"user"`
	LastMessage    *MessagePreview `json:"lastMessage,omitempty"`
	UnreadCount    int             `json:"unreadCount"`
}

type ChatsResponse struct {
	Chats       []ChatSummary `json:"chats"`
	TotalUnread int           `json:"totalUnread"`
}

type Message struct {
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	HasMore         bool       `json:"hasMore"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	UserID         string    `json:"userId"`
	ReadAt         time.Time `json:"readAt"`
}

type storedSummary struct {
	ChatSummary
	PhotoID *string
	PhotoW  *int
	PhotoH  *int
}

// thread is the access-checked conversation between two matched members.
type thread struct {
	ConversationID  string
	ClearedAt       *time.Time
	OtherLastReadAt *time.Time
}
