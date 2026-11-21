package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type Match struct {
	ID        string
	UserID    string
	OtherID   string
	CreatedAt time.Time
}

type SwipeResult struct {
	Matched bool
	Match   *Match
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,max=36"`
	Action       string `json:"action" binding:"required,max=10"`
}

type MatchResponse struct {
	ID        string                   `json:"id"`
	User      profiles.SummaryResponse `json:"user"`
	CreatedAt time.Time                `json:"createdAt"`
}

type SwipeResponse struct {
	Action  string         `json:"action"`
	Matched bool           `json:"matched"`
	Match   *MatchResponse `json:"match,omitempty"`
}
