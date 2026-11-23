package chat

import "time"

const (
	MaxMessageRunes     = 2000
	DefaultMessagesPage = 30
	MaxMessagesPage     = 100
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderUserID   string    `json:"senderUserId"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Membership describes the caller's access to a conversation.
type Membership struct {
	ConversationID  string
	OtherUserID     string
	OtherLastReadAt *time.Time
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	HasMore         bool       `json:"hasMore"`
	OtherUserID     string     `json:"otherUserId"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ReadReceipt struct {
	ConversationID string    `json:"conversationId"`
	UserID         string    `json:"userId"`
	LastReadAt     time.Time `json:"lastReadAt"`
}
