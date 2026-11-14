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
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type Match struct {
	ID        string                 `json:"id"`
	User      profiles.PublicProfile `json:"user"`
	CreatedAt time.Time              `json:"createdAt"`
}

type SwipeResponse struct {
	Matched bool   `json:"matched"`
	Match   *Match `json:"match,omitempty"`
}

type DiscoveryResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type swipeResult struct {
	MatchID        string
	MatchCreatedAt time.Time
}
