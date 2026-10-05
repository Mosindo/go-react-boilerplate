package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	defaultDiscoverLimit = 10
	maxDiscoverLimit     = 30
	defaultMatchesLimit  = 30
	maxMatchesLimit      = 100
)

type DiscoverResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,max=36"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type MatchPreview struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	CreatedAt      time.Time       `json:"createdAt"`
	User           profiles.Card   `json:"user"`
	LastMessage    *MessagePreview `json:"lastMessage,omitempty"`
	UnreadCount    int             `json:"unreadCount"`
}

type MessagePreview struct {
	Content      string    `json:"content"`
	SenderUserID string    `json:"senderUserId"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SwipeResponse struct {
	Action  string        `json:"action"`
	Matched bool          `json:"matched"`
	Match   *MatchPreview `json:"match,omitempty"`
}

type MatchesResponse struct {
	Matches []MatchPreview `json:"matches"`
}
