package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoverLimit = 10
	MaxDiscoverLimit     = 20
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

// MatchSummary is returned when a swipe creates a match and by GET /matches.
type MatchSummary struct {
	ID             string                 `json:"id"`
	CreatedAt      time.Time              `json:"createdAt"`
	ConversationID string                 `json:"conversationId"`
	HasMessages    bool                   `json:"hasMessages"`
	User           profiles.PublicProfile `json:"user"`
}

type SwipeResponse struct {
	Action        string        `json:"action"`
	Matched       bool          `json:"matched"`
	AlreadySwiped bool          `json:"alreadySwiped"`
	Match         *MatchSummary `json:"match,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type MatchesResponse struct {
	Matches []MatchSummary `json:"matches"`
}

// SwipeResult is the repository outcome of a swipe.
type SwipeResult struct {
	Action         string
	Inserted       bool
	MatchID        string
	ConversationID string
	MatchedAt      time.Time
}

// MatchRow is the repository shape of a match before profiles are attached.
type MatchRow struct {
	ID             string
	CreatedAt      time.Time
	OtherUserID    string
	ConversationID string
	HasMessages    bool
}
