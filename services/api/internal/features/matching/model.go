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
	ErrProfileIncomplete = errors.New("complete your profile to discover people")
	ErrInvalidDecision   = errors.New("decision must be like or pass")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
	ErrTargetUnavailable = errors.New("this profile is no longer available")
	ErrAlreadySwiped     = errors.New("you already responded to this profile")
	ErrMatchNotFound     = errors.New("match not found")
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Decision     string `json:"decision" binding:"required"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type SwipeResponse struct {
	Decision string `json:"decision"`
	Matched  bool   `json:"matched"`
	Match    *Match `json:"match,omitempty"`
}

type Match struct {
	ID            string                 `json:"id"`
	User          profiles.PublicProfile `json:"user"`
	CreatedAt     time.Time              `json:"createdAt"`
	LastMessageAt *time.Time             `json:"lastMessageAt,omitempty"`
}

type MatchesResponse struct {
	Matches []Match `json:"matches"`
}

type viewerContext struct {
	OrganizationID string
	Gender         string
	Birthdate      time.Time
	Latitude       *float64
	Longitude      *float64
	InterestedIn   []string
	MinAge         int
	MaxAge         int
	MaxDistanceKm  int
}

type discoverQuery struct {
	ViewerID       string
	OrganizationID string
	Gender         string
	Age            int
	InterestedIn   []string
	Earliest       time.Time
	Latest         time.Time
	Latitude       *float64
	Longitude      *float64
	DeltaLat       *float64
	DeltaLng       *float64
	MaxDistanceKm  int
	Exclude        []string
	Limit          int
}

type swipeResult struct {
	Matched        bool
	MatchID        string
	ConversationID string
	CreatedAt      time.Time
}

type storedMatch struct {
	ID            string
	OtherUserID   string
	CreatedAt     time.Time
	LastMessageAt *time.Time
}
