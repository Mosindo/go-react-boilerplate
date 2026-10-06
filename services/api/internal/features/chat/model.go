package chat

import (
	"errors"
	"time"
)

const (
	defaultMatchesLimit  = 30
	maxMatchesLimit      = 100
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100
	maxMessageRunes      = 2000
)

var (
	ErrMatchNotFound = errors.New("match not found")
	ErrEmptyMessage  = errors.New("message required")
	ErrMessageTooBig = errors.New("message too long")
)

type Peer struct {
	UserID       string
	FirstName    string
	PrimaryPhoto string
}

type Message struct {
	ID        string
	MatchID   string
	SenderID  string
	Body      string
	CreatedAt time.Time
}

type MatchSummary struct {
	ID          string
	Peer        Peer
	CreatedAt   time.Time
	LastMessage *Message
	UnreadCount int
}

// Membership proves a user belongs to an active match and identifies the other side.
type Membership struct {
	MatchID        string
	ConversationID string
	OtherUserID    string
	OtherLastRead  *time.Time
	SelfName       string
}

// --- HTTP payloads ---

type PeerResponse struct {
	UserID       string `json:"userId"`
	FirstName    string `json:"firstName"`
	PrimaryPhoto string `json:"primaryPhotoUrl,omitempty"`
}

type MessageResponse struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"matchId"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type MatchResponse struct {
	ID          string           `json:"id"`
	User        PeerResponse     `json:"user"`
	CreatedAt   time.Time        `json:"createdAt"`
	LastMessage *MessageResponse `json:"lastMessage,omitempty"`
	UnreadCount int              `json:"unreadCount"`
}

type MatchesResponse struct {
	Matches     []MatchResponse `json:"matches"`
	UnreadTotal int             `json:"unreadTotal"`
}

type MessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	HasMore  bool              `json:"hasMore"`
	// PeerLastReadAt lets the sender render read receipts: any of my messages
	// created at or before this instant has been read.
	PeerLastReadAt *time.Time `json:"peerLastReadAt"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

func ToMessageResponse(m Message) MessageResponse {
	return MessageResponse{ID: m.ID, MatchID: m.MatchID, SenderID: m.SenderID, Body: m.Body, CreatedAt: m.CreatedAt}
}
