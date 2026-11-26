package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	defaultDeckSize = 10
	maxDeckSize     = 20
	candidatePool   = 200
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type MatchView struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	HasMessages    bool                   `json:"hasMessages"`
	Profile        profiles.PublicProfile `json:"profile"`
}

type SwipeResult struct {
	Action  string     `json:"action"`
	Matched bool       `json:"matched"`
	Match   *MatchView `json:"match"`
}

type DeckResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
	// Reason explains an empty deck so the client can show the right empty state.
	Reason string `json:"reason,omitempty"`
}

type MatchesPage struct {
	Matches    []MatchView `json:"matches"`
	NextCursor *string     `json:"nextCursor"`
}

// viewer is the discovery context of the current user.
type viewer struct {
	UserID        string
	Gender        string
	Age           int
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
	Latitude      *float64
	Longitude     *float64
	Complete      bool
}

// Candidate is a discovery row before ranking.
type Candidate struct {
	UserID          string
	LastActiveAt    time.Time
	DistanceKm      *float64
	LikedMe         bool
	SharedInterests int
}

type swipeOutcome struct {
	AlreadySwiped  bool
	Matched        bool
	MatchID        string
	ConversationID string
	CreatedAt      time.Time
}

type matchRow struct {
	ID             string
	OtherUserID    string
	ConversationID string
	CreatedAt      time.Time
	HasMessages    bool
}
