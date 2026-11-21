package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	MaxMessageRunes          = 2000
	DefaultChatsLimit        = 20
	MaxChatsLimit            = 50
	DefaultChatMessagesLimit = 50
	MaxChatMessagesLimit     = 100
)

type Conversation struct {
	ID              string
	MatchID         string
	MyClearedAt     *time.Time
	OtherLastReadAt *time.Time
}

type ChatRow struct {
	MatchID     string
	OtherUserID string
	MatchedAt   time.Time
	LastMessage *ChatMessage
	UnreadCount int
}

type ChatMessage struct {
	ID              string
	SenderUserID    string
	RecipientUserID string
	Content         string
	CreatedAt       time.Time
}

type ChatMessagePreviewResponse struct {
	ID           string    `json:"id"`
	SenderUserID string    `json:"senderUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ChatSummaryResponse struct {
	MatchID     string                      `json:"matchId"`
	User        profiles.SummaryResponse    `json:"user"`
	MatchedAt   time.Time                   `json:"matchedAt"`
	LastMessage *ChatMessagePreviewResponse `json:"lastMessage,omitempty"`
	UnreadCount int                         `json:"unreadCount"`
}

type ChatsResponse struct {
	Chats []ChatSummaryResponse `json:"chats"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ChatMessageResponse struct {
	ID              string    `json:"id"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
	IsRead          bool      `json:"isRead"`
}

type ChatMessagesResponse struct {
	Messages        []ChatMessageResponse `json:"messages"`
	HasMore         bool                  `json:"hasMore"`
	OtherLastReadAt *time.Time            `json:"otherLastReadAt"`
}

type ReadReceiptResponse struct {
	ReadAt time.Time `json:"readAt"`
}
