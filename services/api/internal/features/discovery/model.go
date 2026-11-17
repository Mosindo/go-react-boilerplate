package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 10
	MaxLimit     = 20
	poolFactor   = 5
	maxPool      = 100
)

// Candidate is a pre-filtered profile eligible for the viewer. All hard rules
// (mutual preferences, distance, blocks, previous swipes) are enforced in SQL;
// ranking only orders this set.
type Candidate struct {
	UserID          string
	DistanceKm      float64
	LastActiveAt    time.Time
	SharedInterests int
	LikedViewer     bool
}

type Card struct {
	profiles.PublicProfile
	SharedInterests int `json:"sharedInterests"`
}

type FeedResponse struct {
	Profiles []Card `json:"profiles"`
}

type BoundingBox struct {
	MinLat, MaxLat, MinLng, MaxLng float64
	CheckLongitude                 bool
}
