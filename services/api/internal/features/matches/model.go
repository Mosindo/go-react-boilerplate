package matches

import (
	"time"

	"example.com/api/internal/platform/storage"
)

type SwipeRequest struct {
	TargetID string `json:"targetId"`
	Action   string `json:"action"`
}

// UserSummary is the minimal public identity shown in lists.
type UserSummary = storage.UserSummary

type Match struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversationId"`
	CreatedAt      time.Time   `json:"createdAt"`
	User           UserSummary `json:"user"`
}

type SwipeResult struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	Match   *Match `json:"match,omitempty"`
}

type MatchPage struct {
	Items      []Match `json:"items"`
	NextBefore *string `json:"nextBefore"`
}
