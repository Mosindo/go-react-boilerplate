package discovery

import (
	"time"

	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 10
	MaxLimit     = 50

	ActionLike = "like"
	ActionPass = "pass"
)

// Candidate is what other users may see of a profile. It never carries the
// birth date, exact coordinates or exact distance.
type Candidate struct {
	UserID              string              `json:"userId"`
	FirstName           string              `json:"firstName"`
	Age                 int                 `json:"age"`
	Bio                 string              `json:"bio"`
	DistanceKm          *int                `json:"distanceKm"`
	LocationLabel       string              `json:"locationLabel"`
	Interests           []profiles.Interest `json:"interests"`
	SharedInterestCount int                 `json:"sharedInterestCount"`
	Photos              []profiles.Photo    `json:"photos"`
}

type DiscoverResponse struct {
	Items []Candidate `json:"items"`
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId"`
	Action       string `json:"action"`
}

type SwipeResponse struct {
	Matched bool             `json:"matched"`
	Match   *matches.Summary `json:"match,omitempty"`
}

// Viewer is the requesting user's discovery context, read from the database
// (never from the client).
type Viewer struct {
	UserID       string
	Gender       string
	BirthDate    time.Time
	Lat, Lon     float64
	Prefs        profiles.Preferences
	HasProfile   bool
	HasLocation  bool
	HasPhoto     bool
	Discoverable bool
}

func (v Viewer) Complete() bool { return v.HasProfile && v.HasLocation && v.HasPhoto }

// CandidateRow is a raw candidate before batching interests/photos.
type CandidateRow struct {
	UserID          string
	FirstName       string
	BirthDate       time.Time
	Bio             string
	LocationLabel   string
	ShowDistance    bool
	LastActiveAt    time.Time
	DistanceKm      *float64
	SharedInterests int
}

// SwipeOutcome is the result of the swipe transaction.
type SwipeOutcome struct {
	Action        string
	Matched       bool
	MatchID       string
	Created       bool // this call created the match
	Notifications map[string]notificationPayload
}
