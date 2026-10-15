package matching

import "example.com/api/internal/features/users"

const (
	ActionLike = "like"
	ActionPass = "pass"

	defaultDiscoverLimit = 10
	maxDiscoverLimit     = 20
	maxExclude           = 50
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type MatchInfo struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	UserID         string `json:"userId"`
	FirstName      string `json:"firstName"`
}

type SwipeResponse struct {
	Matched bool       `json:"matched"`
	Match   *MatchInfo `json:"match,omitempty"`
}

type DiscoverResponse struct {
	Profiles []users.Card `json:"profiles"`
}

// SwipeResult is what the repository reports after recording a decision.
type SwipeResult struct {
	Matched        bool
	MatchID        string
	ConversationID string
	ViewerName     string
	TargetName     string
}
