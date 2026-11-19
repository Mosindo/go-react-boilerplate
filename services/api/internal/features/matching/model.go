package matching

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DecisionLike = "like"
	DecisionPass = "pass"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (with at least one photo) to start discovering")
	ErrTargetUnavailable = errors.New("profile not available")
	ErrAlreadySwiped     = errors.New("you already answered this profile")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
	ErrMatchNotFound     = errors.New("match not found")
)

// Viewer is the discovery context of the requesting user.
type Viewer struct {
	UserID        string
	Gender        string
	Birthdate     time.Time
	Latitude      *float64
	Longitude     *float64
	InterestedIn  []string
	AgeMin        int
	AgeMax        int
	MaxDistanceKm int
	HasPhoto      bool
}

// CandidateQuery describes the filters applied by the repository.
type CandidateQuery struct {
	Viewer      Viewer
	Now         time.Time
	Limit       int
	IncludeDemo bool
}

type SwipeResult struct {
	Matched        bool
	MatchID        string
	ConversationID string
}

type SwipeResponse struct {
	Matched bool           `json:"matched"`
	Match   *MatchResponse `json:"match,omitempty"`
}

type MatchResponse struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	User           profiles.Card `json:"user"`
}

type DiscoveryResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}
