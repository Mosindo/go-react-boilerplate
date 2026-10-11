package chat

import "time"

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationUser struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Conversation struct {
	ID             string           `json:"id"`
	MatchID        string           `json:"matchId"`
	User           ConversationUser `json:"user"`
	LastMessage    *LastMessage     `json:"lastMessage"`
	UnreadCount    int              `json:"unreadCount"`
	CreatedAt      time.Time        `json:"createdAt"`
	LastActivityAt time.Time        `json:"lastActivityAt"`
}

type SendRequest struct {
	Body string `json:"body" binding:"required"`
}

type ConversationsPage struct {
	Conversations []Conversation `json:"conversations"`
	NextCursor    *string        `json:"nextCursor"`
}

type MessagesPage struct {
	Messages        []Message  `json:"messages"`
	NextCursor      *string    `json:"nextCursor"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}
