package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

// Viewer is the subset of the requesting member's profile used for filtering.
type Viewer struct {
	UserID         string
	OrganizationID string
	Gender         string
	Age            int
	Latitude       *float64
	Longitude      *float64
	InterestedIn   []string
	MinAge         int
	MaxAge         int
	MaxDistanceKm  int
}

// Candidate is a member who passed every hard filter (mutual preferences,
// distance, not blocked, not already swiped). Ranking happens on these.
type Candidate struct {
	UserID          string
	DistanceKm      *float64
	LikedViewer     bool
	SharedInterests int
	PhotoCount      int
	HasBio          bool
	LastActiveAt    time.Time
	CreatedAt       time.Time
}

type CandidateQuery struct {
	Viewer     Viewer
	ExcludeIDs []string
	PoolSize   int
	LatDelta   float64
	LngDelta   float64
}

type FeedItem struct {
	profiles.CardResponse
	LikedYou        bool `json:"likedYou"`
	SharedInterests int  `json:"sharedInterests"`
}

type FeedResponse struct {
	Profiles []FeedItem `json:"profiles"`
}
