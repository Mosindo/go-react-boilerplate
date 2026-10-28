package chat

import "time"

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type Peer struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
}

// Conversation is the chat view of a match; its id is the match id.
type Conversation struct {
	ID          string    `json:"id"`
	MatchedAt   time.Time `json:"matchedAt"`
	User        Peer      `json:"user"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
	ActivityAt  time.Time `json:"-"`
}

type ConversationPage struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagePage struct {
	Messages   []Message `json:"messages"`
	NextCursor *string   `json:"nextCursor"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

// access describes how the caller relates to a conversation.
type access struct {
	PeerID        string
	BlockedByMe   bool
	BlockedByPeer bool
}

type ReadReceipt struct {
	ConversationID string    `json:"conversationId"`
	ReaderID       string    `json:"readerId"`
	ReadAt         time.Time `json:"readAt"`
}
