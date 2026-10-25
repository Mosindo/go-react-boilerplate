package discovery

import (
	"errors"
	"time"

	"example.com/api/internal/features/matches"
)

type Action string

const (
	ActionLike Action = "like"
	ActionPass Action = "pass"
)

const (
	ResultLiked         = "liked"
	ResultPassed        = "passed"
	ResultMatched       = "matched"
	ResultAlreadySwiped = "already_swiped"
	DefaultFeedSize     = 10
	MaxFeedSize         = 20
)

var (
	ErrTargetUnavailable = errors.New("profile not available")
	ErrProfileIncomplete = errors.New("complete your profile first")
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required"`
	Action Action `json:"action" binding:"required"`
}

type SwipeResponse struct {
	Result string        `json:"result"`
	Match  *matches.Item `json:"match"`
}

// outcome is what the repository reports after the swipe transaction commits.
type outcome struct {
	Result         string
	MatchID        string
	ConversationID string
	MatchCreatedAt time.Time
	// TargetNotificationID is the unread "match" notification created for the other user.
	TargetNotificationID string
}
