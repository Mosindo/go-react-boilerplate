package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	undoWindow = 10 * time.Minute
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type Match struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	OtherUserID    string    `json:"otherUserId"`
	CreatedAt      time.Time `json:"createdAt"`
	// InitiatedByMe is true for the member whose like completed the match; their client
	// already shows the match screen and skips the realtime toast.
	InitiatedByMe bool `json:"initiatedByMe"`
}

type MatchWithProfile struct {
	Match
	Profile profiles.PublicProfile `json:"profile"`
}

type SwipeResponse struct {
	Action  string            `json:"action"`
	Matched bool              `json:"matched"`
	Match   *MatchWithProfile `json:"match"`
}

type UndoResponse struct {
	Profile profiles.PublicProfile `json:"profile"`
}

// SwipeOutcome is what the repository reports after the swipe transaction.
type SwipeOutcome struct {
	Matched        bool
	MatchID        string
	ConversationID string
	MatchedAt      time.Time
}
