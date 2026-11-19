package chat

import (
	"errors"
	"time"
)

const MaxMessageLength = 2000

var (
	ErrNotFound       = errors.New("conversation not found")
	ErrClosed         = errors.New("this conversation is closed")
	ErrInvalidMessage = errors.New("messages must contain 1 to 2000 characters")
	ErrInvalidCursor  = errors.New("unknown cursor")
)

// Membership is the access-control view of a conversation for one participant.
type Membership struct {
	ConversationID    string
	MatchID           string
	PartnerID         string
	Active            bool // match still active and no block in either direction
	ClearedAt         *time.Time
	PartnerLastReadAt *time.Time
	MatchedAt         time.Time
}

type StoredMessage struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	CreatedAt      time.Time
}

type StoredSummary struct {
	ConversationID    string
	MatchID           string
	PartnerID         string
	MatchedAt         time.Time
	ActivityAt        time.Time
	LastMessage       *StoredMessage
	UnreadCount       int
	PartnerLastReadAt *time.Time
}

type Partner struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	IsMine         bool      `json:"isMine"`
	IsRead         bool      `json:"isRead"`
}

type Conversation struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"matchId"`
	MatchedAt   time.Time `json:"matchedAt"`
	Partner     Partner   `json:"partner"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesResponse struct {
	Messages          []Message  `json:"messages"`
	NextCursor        *string    `json:"nextCursor"`
	PartnerLastReadAt *time.Time `json:"partnerLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required"`
}
