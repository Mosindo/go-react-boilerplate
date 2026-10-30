package chat

import "time"

const (
	MaxMessageRunes     = 2000
	DefaultPageSize     = 30
	MaxPageSize         = 100
	DefaultConversation = 30
)

type Participant struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	PhotoURL    string `json:"photoUrl,omitempty"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	ReadByOther    bool      `json:"readByOther"`
}

type Conversation struct {
	ID             string      `json:"id"`
	MatchID        string      `json:"matchId"`
	User           Participant `json:"user"`
	LastMessage    *Message    `json:"lastMessage,omitempty"`
	UnreadCount    int         `json:"unreadCount"`
	LastActivityAt time.Time   `json:"lastActivityAt"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	NextBefore    *time.Time     `json:"nextBefore,omitempty"`
}

type MessagesResponse struct {
	Messages   []Message  `json:"messages"`
	NextBefore *time.Time `json:"nextBefore,omitempty"`
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type storedConversation struct {
	ID             string
	MatchID        string
	OtherUserID    string
	OtherName      string
	OtherPhotoID   *string
	LastMessage    *Message
	UnreadCount    int
	LastActivityAt time.Time
}
