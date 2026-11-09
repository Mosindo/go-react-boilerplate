package matching

import "errors"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultPageSize = 10
	MaxPageSize     = 20
)

var (
	ErrSelf              = errors.New("cannot swipe yourself")
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrIncompleteProfile = errors.New("complete your profile before swiping")
	ErrInvalidAction     = errors.New("action must be like or pass")
	ErrMatchNotFound     = errors.New("match not found")
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type SwipeResult struct {
	Action         string `json:"action"`
	Matched        bool   `json:"matched"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	// AlreadySwiped is true when this exact decision had been recorded before (idempotent replay).
	AlreadySwiped bool `json:"alreadySwiped"`
}

// MatchEvent is what the repository returns when a new match was created.
type MatchEvent struct {
	MatchID        string
	ConversationID string
	UserA          string
	UserB          string
}
