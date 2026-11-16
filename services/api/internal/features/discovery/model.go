package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

// Candidate is a profile that passed every hard filter (preferences, distance,
// blocks, already-swiped). Ranking only orders candidates; it never widens
// what a user is allowed to see.
type Candidate struct {
	UserID          string
	LastActiveAt    time.Time
	LikedViewer     bool
	SharedInterests int
	DistanceKm      *float64
	MaxDistanceKm   int
}

type Response struct {
	Profiles []profiles.Card `json:"profiles"`
	// Reason explains an empty deck so the client can guide the user.
	Reason string `json:"reason,omitempty"`
}
