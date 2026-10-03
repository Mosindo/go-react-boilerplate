package conversations

import "time"

const (
	MaxMessageRunes = 2000
	DefaultPageSize = 30
	MaxPageSize     = 50
)

type Participant struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	PhotoURL  string `json:"photoUrl,omitempty"`
}

type Message struct {
	ID        string     `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	MatchID     string       `json:"matchId"`
	User        Participant  `json:"user"`
	LastMessage *LastMessage `json:"lastMessage"`
	UnreadCount int          `json:"unreadCount"`
	MatchedAt   time.Time    `json:"matchedAt"`
	SortKey     time.Time    `json:"updatedAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
}
