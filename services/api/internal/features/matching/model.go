package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DecisionLike = "like"
	DecisionPass = "pass"

	DefaultDiscoveryLimit = 10
	MaxDiscoveryLimit     = 30
	candidatePoolSize     = 200
)

// Candidate is a discoverable profile plus the signals used for ranking.
type Candidate struct {
	UserID       string
	DistanceKm   float64
	SharedCount  int
	LikedViewer  bool
	LastActiveAt time.Time
}

type Match struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Decision     string `json:"decision" binding:"required,oneof=like pass"`
}

type SwipeResponse struct {
	Decision string         `json:"decision"`
	Matched  bool           `json:"matched"`
	Match    *MatchResponse `json:"match,omitempty"`
}

type MatchResponse struct {
	Match
	User profiles.Card `json:"user"`
}

type DiscoveryResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}
