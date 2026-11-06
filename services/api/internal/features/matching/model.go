package matching

import "example.com/api/internal/features/profiles"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultBatch = 10
	MaxBatch     = 20
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required,max=36"`
	Action   string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResponse struct {
	Matched        bool   `json:"matched"`
	Duplicate      bool   `json:"duplicate"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

// Candidate is a raw recommendation row before photos and interests are attached.
type Candidate struct {
	ID           string
	FirstName    string
	Age          int
	Gender       string
	Bio          string
	City         string
	DistanceKm   float64
	ShowDistance bool
}

// SwipeResult is the storage-level outcome of a swipe.
type SwipeResult struct {
	Duplicate      bool
	Matched        bool
	MatchID        string
	ConversationID string
	ActorName      string
	TargetName     string
}
