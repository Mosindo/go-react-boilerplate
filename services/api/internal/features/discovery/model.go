package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type MatchView struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	User           profiles.Card `json:"user"`
}

type SwipeResponse struct {
	Action  string     `json:"action"`
	Matched bool       `json:"matched"`
	Match   *MatchView `json:"match,omitempty"`
}

// Counterparts of a stored match used internally.
type storedMatch struct {
	ID             string
	ConversationID string
	OtherUserID    string
	CreatedAt      time.Time
}
