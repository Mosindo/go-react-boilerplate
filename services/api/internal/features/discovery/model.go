package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDeckSize = 10
	MaxDeckSize     = 30
	// CandidatePool is how many pre-filtered rows are scored per request.
	CandidatePool = 200
	MaxExcluded   = 60
)

// Viewer is the requesting user's matching context.
type Viewer struct {
	UserID        string
	Gender        string
	Birthdate     time.Time
	Latitude      float64
	Longitude     float64
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
	PhotoCount    int
	Suspended     bool
}

// Candidate is a pre-filtered profile with the signals used for ranking.
type Candidate struct {
	UserID          string
	Latitude        float64
	Longitude       float64
	LastActiveAt    time.Time
	LikedViewer     bool
	SharedInterests int
	DistanceKm      float64
	Score           float64
}

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required,uuid"`
	Action   string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResult struct {
	Matched bool
	MatchID string
}

type DeckResponse struct {
	Profiles []profiles.CardResponse `json:"profiles"`
}

type SwipeResponse struct {
	Matched bool    `json:"matched"`
	MatchID *string `json:"matchId"`
}
