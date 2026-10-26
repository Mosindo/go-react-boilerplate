package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultBatch = 10
	MaxBatch     = 20
)

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type MatchView struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	Profile   profiles.PublicProfile `json:"profile"`
}

type SwipeResponse struct {
	Matched       bool       `json:"matched"`
	AlreadySwiped bool       `json:"alreadySwiped"`
	Match         *MatchView `json:"match,omitempty"`
}
