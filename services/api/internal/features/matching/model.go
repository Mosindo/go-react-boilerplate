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

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type MatchInfo struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	User           profiles.PublicProfile `json:"user"`
}

type SwipeResponse struct {
	Matched bool       `json:"matched"`
	Match   *MatchInfo `json:"match,omitempty"`
}

type ResetPassesResponse struct {
	Reset int64 `json:"reset"`
}

// candidate is a discovery row before ranking.
type candidate struct {
	Card            profiles.Card
	DistanceKm      float64
	ShowDistance    bool
	SharedInterests int
}

type swipeOutcome struct {
	Matched        bool
	MatchID        string
	ConversationID string
	CreatedAt      time.Time
	ViewerName     string
	TargetName     string
}
