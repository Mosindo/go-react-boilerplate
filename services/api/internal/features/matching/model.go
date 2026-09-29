package matching

import (
	"errors"
	"time"
)

var (
	ErrBadRequest        = errors.New("invalid request")
	ErrSelfSwipe         = errors.New("cannot swipe yourself")
	ErrNotEligible       = errors.New("profile not found")
	ErrDuplicateSwipe    = errors.New("already swiped")
	ErrIncompleteProfile = errors.New("complete your profile first")
	ErrMatchNotFound     = errors.New("match not found")
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type swipeRequest struct {
	UserID string `json:"userId"`
	Action string `json:"action"`
}

// Photo, SummaryUser and ConversationSummary mirror the shapes in docs/API.md.
type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type SummaryUser struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	Age       *int   `json:"age"`
	Photo     *Photo `json:"photo"`
}

type ConversationSummary struct {
	ID          string      `json:"id"`
	MatchID     string      `json:"matchId"`
	MatchedAt   time.Time   `json:"matchedAt"`
	User        SummaryUser `json:"user"`
	LastMessage any         `json:"lastMessage"`
	UnreadCount int         `json:"unreadCount"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type SwipeResponse struct {
	Matched      bool                 `json:"matched"`
	Conversation *ConversationSummary `json:"conversation"`
}

// swipeOutcome is what the transaction reports back to the service.
type swipeOutcome struct {
	Matched        bool
	MatchID        string
	ConversationID string
	MatchedAt      time.Time
}

// party is the public bit of a user needed to build a conversation summary.
type party struct {
	UserID    string
	FirstName string
	Age       *int
	PhotoID   *string
}
