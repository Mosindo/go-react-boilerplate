package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	defaultDiscoveryLimit = 10
	maxDiscoveryLimit     = 20
	// candidatePoolSize bounds the SQL prefilter; ranking happens in Go on
	// this pool, then only the returned page is hydrated.
	candidatePoolSize = 200
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type MatchInfo struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	User      profiles.PublicProfile `json:"user"`
}

type SwipeResult struct {
	Action  string     `json:"action"`
	Matched bool       `json:"matched"`
	Match   *MatchInfo `json:"match,omitempty"`
}

type DiscoveryResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

// Candidate is the lightweight row used for ranking.
type Candidate struct {
	UserID          string
	DistanceKm      float64
	MaxDistanceKm   float64
	LastActiveAt    time.Time
	SharedInterests int
}

type swipeOutcome struct {
	Matched   bool
	MatchID   string
	CreatedAt time.Time
	New       bool
}
