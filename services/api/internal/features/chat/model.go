package chat

import "time"

const (
	MaxMessageRunes      = 2000
	MaxClientIDLength    = 64
	DefaultMessagesLimit = 30
	MaxMessagesLimit     = 100
)

type Conversation struct {
	ID              string
	MatchID         string
	OtherUserID     string
	OtherLastReadAt *time.Time
}

type Message struct {
	ID           string
	SenderUserID string
	Content      string
	ClientID     string
	CreatedAt    time.Time
}

type SendMessageRequest struct {
	Content  string `json:"content" binding:"required"`
	ClientID string `json:"clientId"`
}

type MessageResponse struct {
	ID           string    `json:"id"`
	MatchID      string    `json:"matchId"`
	SenderUserID string    `json:"senderUserId"`
	Content      string    `json:"content"`
	ClientID     string    `json:"clientId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	IsRead       bool      `json:"isRead"`
}

type MessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	// OtherLastReadAt lets the client render "seen" receipts.
	OtherLastReadAt *time.Time `json:"otherLastReadAt,omitempty"`
	NextCursor      string     `json:"nextCursor,omitempty"`
}

type ReadReceipt struct {
	MatchID string    `json:"matchId"`
	UserID  string    `json:"userId"`
	ReadAt  time.Time `json:"readAt"`
}
