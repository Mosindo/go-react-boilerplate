package discovery

import (
	"example.com/api/internal/features/chat"
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

type SwipeResult struct {
	Action       string             `json:"action"`
	Matched      bool               `json:"matched"`
	Conversation *chat.Conversation `json:"conversation,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

// swipeOutcome is the repository-level result of recording a swipe.
type swipeOutcome struct {
	Action  string
	MatchID string // empty when there is no match
	Created bool   // true when this call created the match
}
