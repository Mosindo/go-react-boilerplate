package chat

import "time"

const (
	MaxBodyRunes = 2000
)

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	// Read is true for messages the other participant has read (own messages only).
	Read bool `json:"read"`
}

type Peer struct {
	ID          string  `json:"id"`
	FirstName   string  `json:"firstName"`
	MainPhotoID *string `json:"mainPhotoId"`
}

type MessagePreview struct {
	ID        int64     `json:"id"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string          `json:"id"`
	MatchID     string          `json:"matchId"`
	Peer        Peer            `json:"peer"`
	LastMessage *MessagePreview `json:"lastMessage"`
	UnreadCount int             `json:"unreadCount"`
	MatchedAt   time.Time       `json:"matchedAt"`
	Activity    time.Time       `json:"activityAt"`
}

type ConversationPage struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

// Events lets realtime / notifications react to chat activity.
type Events interface {
	MessageCreated(msg Message, senderName, recipientID string)
	MessagesRead(conversationID, readerID, otherID string, upTo int64)
}
