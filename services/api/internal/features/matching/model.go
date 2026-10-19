package matching

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DiscoverDefaultLimit = 10
	DiscoverMaxLimit     = 20
	ListDefaultLimit     = 20
	ListMaxLimit         = 50
)

var (
	ErrProfileIncomplete = errors.New("complete your profile, preferences and add a photo first")
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrInvalidAction     = errors.New("invalid action")
	ErrMatchNotFound     = errors.New("match not found")
	ErrInvalidCursor     = errors.New("invalid cursor")
)

// Viewer holds the attributes of the requesting user that candidate selection
// depends on. It is loaded server-side; the client never supplies it.
type Viewer struct {
	UserID        string
	Gender        string
	Age           int
	Lat, Lng      *float64
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm *int
}

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required"`
}

type Match struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	Profile        profiles.PublicProfile `json:"profile"`
}

type SwipeResult struct {
	Matched   bool   `json:"matched"`
	Duplicate bool   `json:"duplicate"`
	Match     *Match `json:"match,omitempty"`
}

// matchCreated is returned by the repository so the service can fan out
// realtime events after the transaction commits.
type matchCreated struct {
	MatchID        string
	ConversationID string
	CreatedAt      time.Time
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type MatchesResponse struct {
	Matches    []Match `json:"matches"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

type LikesResponse struct {
	Profiles   []LikedBy `json:"profiles"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type LikedBy struct {
	profiles.PublicProfile
	LikedAt time.Time `json:"likedAt"`
}
