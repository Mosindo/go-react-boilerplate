package matching

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile before swiping")
	ErrTargetUnavailable = errors.New("this profile is not available")
	ErrAlreadySwiped     = errors.New("you already answered this profile")
	ErrSelfSwipe         = errors.New("you cannot swipe on yourself")
	ErrMatchNotFound     = errors.New("match not found")
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type MatchView struct {
	ID             string        `json:"id"`
	MatchedAt      time.Time     `json:"matchedAt"`
	ConversationID string        `json:"conversationId,omitempty"`
	User           profiles.Card `json:"user"`
}

type SwipeResponse struct {
	Action  string     `json:"action"`
	Matched bool       `json:"matched"`
	Match   *MatchView `json:"match,omitempty"`
}

type DiscoveryResponse struct {
	Profiles []profiles.Card `json:"profiles"`
	// Status explains an empty feed: "ok", "incomplete_profile", "paused"
	// or "under_review".
	Status        string   `json:"status"`
	MissingFields []string `json:"missingFields,omitempty"`
}

type CardsResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

type MatchesResponse struct {
	Matches []MatchView `json:"matches"`
}

type ResetResponse struct {
	Restored int `json:"restored"`
}

// Candidate is a discovery pool entry with the signals used for ranking.
type Candidate struct {
	UserID          string
	DistanceKm      float64
	LastActiveAt    time.Time
	SharedInterests int
	LikedViewer     bool
}

type storedMatch struct {
	ID             string
	OtherUserID    string
	ConversationID *string
	CreatedAt      time.Time
}

type swipeResult struct {
	Action string
	Match  *storedMatch
	// NewLike is true when this call created a like (vs an idempotent replay).
	NewLike  bool
	NewMatch bool
}
