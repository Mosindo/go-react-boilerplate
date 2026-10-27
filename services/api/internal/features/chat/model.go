package chat

import "time"

const (
	MaxMessageRunes = 2000
)

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Peer struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	PhotoID   string `json:"photoId,omitempty"`
}

type ConversationSummary struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"matchId"`
	Peer        Peer      `json:"peer"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ConversationPage struct {
	Conversations []ConversationSummary `json:"conversations"`
	NextCursor    string                `json:"nextCursor,omitempty"`
}

// MessagePage lists messages newest first. OtherLastReadID lets the client show
// which of the caller's messages the other participant has read.
type MessagePage struct {
	Messages        []Message `json:"messages"`
	NextCursor      string    `json:"nextCursor,omitempty"`
	OtherLastReadID int64     `json:"otherLastReadId"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ReadResult struct {
	UpToMessageID int64 `json:"upToMessageId"`
}
