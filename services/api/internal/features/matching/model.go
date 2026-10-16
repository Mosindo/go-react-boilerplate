package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// SwipeResult tells the client whether the swipe produced (or already had) a match.
type SwipeResult struct {
	Matched bool   `json:"matched"`
	MatchID string `json:"matchId,omitempty"`
	// Created is true only for the swipe that formed the match; it drives notifications.
	Created bool   `json:"-"`
	UserA   string `json:"-"`
	UserB   string `json:"-"`
}

type StoredMatch struct {
	ID        string
	OtherID   string
	CreatedAt time.Time
}

type MatchResponse struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	Profile   profiles.PublicProfile `json:"profile"`
}
