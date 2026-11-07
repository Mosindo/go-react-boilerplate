package chat

import "time"

const MaxMessageRunes = 2000

type SendRequest struct {
	Body string `json:"body"`
}

type Message struct {
	ID             int64     `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Peer struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type ConversationSummary struct {
	ID          string    `json:"id"`
	MatchID     string    `json:"matchId"`
	Peer        Peer      `json:"peer"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
	CreatedAt   time.Time `json:"createdAt"`
	SortAt      time.Time `json:"sortAt"`
}

type MessagesPage struct {
	Messages []Message `json:"messages"` // oldest first within the page
	// NextBefore is the id to pass as ?before= to load older messages; null when no more.
	NextBefore *int64 `json:"nextBefore"`
	// PeerReadAt: messages created at or before this instant were read by the other person.
	PeerReadAt *time.Time `json:"peerReadAt"`
	Peer       Peer       `json:"peer"`
}

type storedConversation struct {
	ID        string
	MatchID   string
	PeerID    string
	PeerName  string
	PhotoID   *string
	CreatedAt time.Time
	SortAt    time.Time
	Unread    int
	Last      *Message
}
