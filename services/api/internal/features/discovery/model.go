package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

// Candidate holds the ranking signals of a profile eligible for the viewer.
type Candidate struct {
	UserID          string
	LikedViewer     bool
	SharedInterests int
	DistanceKm      *float64
	LastActiveAt    time.Time
	CreatedAt       time.Time
}

type Response struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}
