package chat

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	FilterAll    = "all"
	FilterNew    = "new"
	FilterActive = "active"

	maxMessageRunes = 2000
)

type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	CreatedAt      time.Time
}

type Conversation struct {
	ID              string
	MatchID         string
	OtherUserID     string
	CreatedAt       time.Time
	LastMessage     *Message
	UnreadCount     int
	OtherLastReadAt *time.Time
	Other           profiles.CardSummary
}

type Membership struct {
	ConversationID  string
	OtherUserID     string
	ClearedAt       *time.Time
	OtherLastReadAt *time.Time
}

type SendMessageRequest struct {
	Body string `json:"body" binding:"required,max=8000"`
}

type MessageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ConversationResponse struct {
	ID              string               `json:"id"`
	MatchID         string               `json:"matchId"`
	User            profiles.CardSummary `json:"user"`
	CreatedAt       time.Time            `json:"createdAt"`
	LastMessage     *MessageResponse     `json:"lastMessage"`
	UnreadCount     int                  `json:"unreadCount"`
	OtherLastReadAt *time.Time           `json:"otherLastReadAt"`
}

type ConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
}

type MessagesResponse struct {
	// Messages are returned newest first; pass the last id as ?before= to page back.
	Messages        []MessageResponse `json:"messages"`
	HasMore         bool              `json:"hasMore"`
	OtherLastReadAt *time.Time        `json:"otherLastReadAt"`
}

type ReadEvent struct {
	ConversationID string    `json:"conversationId"`
	ReaderID       string    `json:"readerId"`
	LastReadAt     time.Time `json:"lastReadAt"`
}

func toMessageResponse(m Message) MessageResponse {
	return MessageResponse{ID: m.ID, ConversationID: m.ConversationID, SenderID: m.SenderID, Body: m.Body, CreatedAt: m.CreatedAt}
}

func toConversationResponse(c Conversation) ConversationResponse {
	resp := ConversationResponse{
		ID:              c.ID,
		MatchID:         c.MatchID,
		User:            c.Other,
		CreatedAt:       c.CreatedAt,
		UnreadCount:     c.UnreadCount,
		OtherLastReadAt: c.OtherLastReadAt,
	}
	if c.LastMessage != nil {
		m := toMessageResponse(*c.LastMessage)
		resp.LastMessage = &m
	}
	return resp
}
