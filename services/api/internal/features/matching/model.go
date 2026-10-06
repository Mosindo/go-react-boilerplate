package matching

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	// ReportsHideThreshold is the number of distinct open reporters after
	// which a profile stops being shown in discovery until reviewed.
	ReportsHideThreshold = 3

	defaultDiscoverLimit = 10
	maxDiscoverLimit     = 20
	candidatePoolSize    = 200
)

var (
	ErrProfileIncomplete = errors.New("complete your profile first")
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrSelfSwipe         = errors.New("cannot swipe yourself")
	ErrInvalidAction     = errors.New("invalid action")
)

// Viewer is the discovering user: what the ranking and filters need to know.
type Viewer struct {
	UserID    string
	Gender    string
	Age       int
	Interests map[int16]struct{}
	Prefs     profiles.Preferences
}

// Candidate is a raw discovery row before interests and photos are attached.
type Candidate struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	City         string
	ShowAge      bool
	ShowDistance bool
	DistanceKm   *float64
	LastActiveAt time.Time
	LikedMe      bool
}

type MatchRef struct {
	ID             string
	ConversationID string
}

type SwipeResult struct {
	Action           string
	AlreadyProcessed bool
	Matched          bool
	NewMatch         bool
	Match            *MatchRef
	// Names are only set for a new match, to build notifications.
	SelfName  string
	OtherName string
}

// --- HTTP payloads ---

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
	Action string `json:"action" binding:"required,max=8"`
}

type SwipeResponse struct {
	Action           string `json:"action"`
	AlreadyProcessed bool   `json:"alreadyProcessed"`
	Matched          bool   `json:"matched"`
	MatchID          string `json:"matchId,omitempty"`
}

type ProfilesResponse struct {
	Profiles []profiles.PublicProfileResponse `json:"profiles"`
}
