package chat

import "time"

const MaxMessageLength = 2000

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Participant struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Conversation struct {
	ID          string      `json:"id"`
	MatchID     string      `json:"matchId"`
	With        Participant `json:"with"`
	LastMessage *Message    `json:"lastMessage"`
	Unread      int         `json:"unread"`
	CreatedAt   time.Time   `json:"createdAt"`
	SortAt      time.Time   `json:"-"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}

type MessagesResponse struct {
	// Messages are returned oldest first.
	Messages []Message `json:"messages"`
	// NextBefore is the id to pass as ?before= to load older messages ("" when done).
	NextBefore string `json:"nextBefore,omitempty"`
	// PeerLastReadID is the id of the last message the other person has read.
	PeerLastReadID string `json:"peerLastReadId"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

type storedConversation struct {
	Conversation
	PhotoID *string
}

type access struct {
	OtherUserID string
	Blocked     bool
	HiddenAt    *time.Time
}
