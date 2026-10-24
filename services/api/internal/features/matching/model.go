package matching

import (
	"errors"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoverLimit = 10
	MaxDiscoverLimit     = 20
)

var (
	ErrNotEligible    = errors.New("profile not available")
	ErrAlreadySwiped  = errors.New("already swiped with a different action")
	ErrProfileMissing = errors.New("complete your profile before discovering")
	ErrMatchNotFound  = errors.New("match not found")
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required"`
	Action   string `json:"action" binding:"required,oneof=like pass"`
}

type MatchInfo struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	User           profiles.Card `json:"user"`
}

type SwipeResult struct {
	Action  string     `json:"action"`
	Matched bool       `json:"matched"`
	Match   *MatchInfo `json:"match,omitempty"`
}

// swipeOutcome is what the repository reports; the service decorates it with profile cards.
type swipeOutcome struct {
	Matched        bool
	NewMatch       bool // created by this call (notifications are only sent then)
	MatchID        string
	ConversationID string
}

type unmatched struct {
	OtherUserID    string
	ConversationID string
}
