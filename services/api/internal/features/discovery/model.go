package discovery

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (with at least one photo) first")
	ErrNotFound          = errors.New("profile not found")
	ErrAlreadySwiped     = errors.New("already swiped")
	ErrInvalidAction     = errors.New("action must be like or pass")
	ErrSelf              = errors.New("cannot swipe yourself")
	ErrMatched           = errors.New("already matched: unmatch instead")
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required"`
	Action string `json:"action" binding:"required"`
}

type SwipeResult struct {
	Action         string `json:"action"`
	Matched        bool   `json:"matched"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	// Match is the other user's card, present only when Matched is true.
	Match *profiles.Card `json:"match,omitempty"`
}

type Match struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	User           profiles.Card `json:"user"`
}

// matchRow is the storage-level match, before cards are attached.
type matchRow struct {
	ID, ConversationID, OtherID string
	CreatedAt                   time.Time
}
