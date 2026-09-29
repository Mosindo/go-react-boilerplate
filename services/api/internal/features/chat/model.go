package chat

import (
	"errors"
	"time"
)

const (
	MaxBodyRunes        = 2000
	DefaultLimit        = 30
	MaxLimit            = 100
	notificationBodyMax = 80
)

var (
	ErrNotFound     = errors.New("conversation not found")
	ErrBlocked      = errors.New("you cannot message this user")
	ErrInvalidBody  = errors.New("message body must be 1-2000 characters")
	ErrInvalidInput = errors.New("invalid input")
)

// Message is the wire shape of a chat message.
type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type ConversationUser struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	Age       *int   `json:"age"`
	Photo     *Photo `json:"photo"`
}

type ConversationSummary struct {
	ID          string           `json:"id"`
	MatchID     string           `json:"matchId"`
	MatchedAt   time.Time        `json:"matchedAt"`
	User        ConversationUser `json:"user"`
	LastMessage *Message         `json:"lastMessage"`
	UnreadCount int              `json:"unreadCount"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type ListConversationsResponse struct {
	Conversations []ConversationSummary `json:"conversations"`
}

type ListMessagesResponse struct {
	Messages   []Message `json:"messages"`
	NextCursor *string   `json:"nextCursor"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

// convoRow is the raw repository row for the conversation list (photo id, not yet signed).
type convoRow struct {
	ID            string
	MatchID       string
	MatchedAt     time.Time
	UpdatedAt     time.Time
	OtherID       string
	FirstName     string
	Age           *int
	PhotoID       *string
	PhotoPosition int
	UnreadCount   int
	LastMessage   *Message
}

// sendContext is what the repository proves inside the send transaction.
type sendResult struct {
	Message         Message
	RecipientID     string
	SenderFirstName string
}
