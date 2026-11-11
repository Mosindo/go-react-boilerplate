package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DirectionLike = "like"
	DirectionPass = "pass"
)

// Candidate holds the ranking features computed by the discovery query.
type Candidate struct {
	UserID          string
	DistanceKm      *float64
	SharedInterests int
	LastActiveAt    time.Time
	LikedViewer     bool
}

type Match struct {
	ID        string
	UserA     string
	UserB     string
	CreatedAt time.Time
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Direction    string `json:"direction" binding:"required,oneof=like pass"`
}

type DiscoverResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

type MatchPayload struct {
	MatchID   string        `json:"matchId"`
	CreatedAt time.Time     `json:"createdAt"`
	User      profiles.Card `json:"user"`
}

type SwipeResponse struct {
	Matched bool          `json:"matched"`
	Match   *MatchPayload `json:"match,omitempty"`
}
