package chat

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound    = errors.New("conversation not found")
	ErrBlocked     = errors.New("conversation is blocked")
	ErrInvalidBody = errors.New("invalid message body")
)

const MaxBodyRunes = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"matchId"`
	UserID      string    `json:"userId"`
	FirstName   string    `json:"firstName"`
	PhotoID     *string   `json:"photoId"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
	OtherReadAt time.Time `json:"otherLastReadAt"`
}

type MessagePage struct {
	Items       []Message `json:"items"`
	NextCursor  string    `json:"nextCursor"`
	OtherReadAt time.Time `json:"otherLastReadAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

// CleanBody normalise et valide le corps d'un message.
func CleanBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" || utf8.RuneCountInString(body) > MaxBodyRunes || strings.ContainsRune(body, 0) {
		return "", ErrInvalidBody
	}
	return body, nil
}

// ReadEvent est pousse en temps reel quand l'autre participant lit la conversation.
type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	ReadAt         time.Time `json:"readAt"`
}
