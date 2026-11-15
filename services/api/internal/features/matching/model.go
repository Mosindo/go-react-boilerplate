package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	// candidatePoolSize is how many eligible profiles SQL pre-selects before
	// the ranker orders them.
	candidatePoolSize = 100
)

// Candidate holds the ranking features computed in SQL for one profile.
type Candidate struct {
	UserID          string
	DistanceKm      *float64
	SharedInterests int
	LikedViewer     bool
	LastActiveAt    time.Time
	PhotoCount      int
	HasBio          bool
}

type ViewerContext struct {
	UserID        string
	MaxDistanceKm int
	Now           time.Time
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type Match struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type SwipeResult struct {
	Matched bool           `json:"matched"`
	Match   *MatchResponse `json:"match"`
}

type MatchResponse struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	User           profiles.Card `json:"user"`
}

type DiscoveryResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

type LikeItem struct {
	LikedAt time.Time     `json:"likedAt"`
	User    profiles.Card `json:"user"`
}

type LikesResponse struct {
	Likes      []LikeItem `json:"likes"`
	NextCursor *string    `json:"nextCursor"`
}

type ReceivedLike struct {
	UserID  string
	LikedAt time.Time
}
