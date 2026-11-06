package chat

import "time"

const (
	MaxBodyRunes = 2000
	MaxPage      = 50
	DefaultPage  = 30
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Peer struct {
	ID          string  `json:"id"`
	FirstName   string  `json:"firstName"`
	MainPhotoID *string `json:"mainPhotoId"`
}

type Conversation struct {
	ID          string       `json:"id"`
	MatchID     string       `json:"matchId"`
	Peer        Peer         `json:"peer"`
	LastMessage *LastMessage `json:"lastMessage"`
	UnreadCount int          `json:"unreadCount"`
	// ActivityAt is the last message time, or the match time for new matches.
	ActivityAt time.Time `json:"activityAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	UnreadTotal   int            `json:"unreadTotal"`
	NextBefore    *time.Time     `json:"nextBefore,omitempty"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	// PeerLastReadAt lets the client show "read" under the messages the other
	// person has seen (message.CreatedAt <= peerLastReadAt).
	PeerLastReadAt time.Time  `json:"peerLastReadAt"`
	Peer           Peer       `json:"peer"`
	NextBefore     *time.Time `json:"nextBefore,omitempty"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

// SendResult is the storage-level outcome of sending a message.
type SendResult struct {
	Message         Message
	RecipientID     string
	SenderName      string
	RecipientUnread int // unread messages from the sender before this one
}
