package chat

import "time"

const MaxMessageRunes = 2000

type Partner struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Conversation is one match as shown in the inbox.
type Conversation struct {
	MatchID     string       `json:"matchId"`
	Partner     Partner      `json:"partner"`
	CreatedAt   time.Time    `json:"createdAt"`
	LastMessage *LastMessage `json:"lastMessage"`
	UnreadCount int          `json:"unreadCount"`
	ActivityAt  time.Time    `json:"activityAt"`
}

type Message struct {
	ID        int64      `json:"id"`
	MatchID   string     `json:"matchId"`
	SenderID  string     `json:"senderId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}
