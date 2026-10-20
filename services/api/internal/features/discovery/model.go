package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,max=36"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type MatchItem struct {
	ID             string               `json:"id"`
	User           profiles.UserSummary `json:"user"`
	ConversationID string               `json:"conversationId"`
	CreatedAt      time.Time            `json:"createdAt"`
}

type SwipeResult struct {
	Matched bool       `json:"matched"`
	Match   *MatchItem `json:"match"`
}

type MatchesResult struct {
	Matches    []MatchItem `json:"matches"`
	NextCursor *string     `json:"nextCursor"`
}

// createdMatch is what the repository returns when a swipe completes a match.
type createdMatch struct {
	MatchID        string
	ConversationID string
}
