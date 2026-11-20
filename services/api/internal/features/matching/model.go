package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

// ViewerContext holds what discovery needs to know about the searching member.
type ViewerContext struct {
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

// Candidate is a discoverable profile with the features used for ranking.
type Candidate struct {
	UserID          string
	DistanceKm      *float64
	SharedInterests int
	IdleHours       float64
	LikedViewer     bool
}

type SwipeResult struct {
	Matched        bool
	MatchID        string
	ConversationID string
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResponse struct {
	Matched        bool                  `json:"matched"`
	MatchID        *string               `json:"matchId"`
	ConversationID *string               `json:"conversationId"`
	MatchedUser    *profiles.CardSummary `json:"matchedUser"`
}

type CardsResponse struct {
	Profiles []profiles.CardResponse `json:"profiles"`
}

// MatchEvent is pushed over the realtime channel to both members.
type MatchEvent struct {
	MatchID        string               `json:"matchId"`
	ConversationID string               `json:"conversationId"`
	User           profiles.CardSummary `json:"user"`
}

type MatchRemovedEvent struct {
	MatchID        string `json:"matchId"`
	ConversationID string `json:"conversationId"`
}
