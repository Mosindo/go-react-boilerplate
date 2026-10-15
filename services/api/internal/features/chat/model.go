package chat

import "time"

const (
	MaxMessageRunes = 2000

	defaultMessagesLimit = 30
	maxMessagesLimit     = 100
	defaultChatsLimit    = 30
	maxChatsLimit        = 50
)

type Peer struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	ThumbURL  *string `json:"thumbUrl,omitempty"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Conversation struct {
	ID          string    `json:"id"`
	Peer        Peer      `json:"peer"`
	LastMessage *Message  `json:"lastMessage,omitempty"`
	UnreadCount int       `json:"unreadCount"`
	MatchedAt   time.Time `json:"matchedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
	// PeerLastReadAt lets the sender show which messages were read.
	PeerLastReadAt *time.Time `json:"peerLastReadAt,omitempty"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type ReadResponse struct {
	ReadAt time.Time `json:"readAt"`
}
