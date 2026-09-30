package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type Match struct {
	ID             string           `json:"id"`
	ConversationID string           `json:"conversationId"`
	CreatedAt      time.Time        `json:"createdAt"`
	User           profiles.Summary `json:"user"`
}

type SwipeResponse struct {
	Matched bool   `json:"matched"`
	Match   *Match `json:"match,omitempty"`
}

type swipeResult struct {
	MatchID        string
	ConversationID string
	MatchedAt      time.Time
}
