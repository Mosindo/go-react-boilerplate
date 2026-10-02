package matches

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required,max=64"`
	Action   string `json:"action" binding:"required,max=10"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type MatchItem struct {
	ID          string        `json:"id"`
	CreatedAt   time.Time     `json:"createdAt"`
	User        profiles.Card `json:"user"`
	LastMessage *LastMessage  `json:"lastMessage,omitempty"`
	UnreadCount int           `json:"unreadCount"`
}

type MatchPage struct {
	Matches    []MatchItem `json:"matches"`
	NextCursor string      `json:"nextCursor,omitempty"`
}

type SwipeResult struct {
	Action  string         `json:"action"`
	Matched bool           `json:"matched"`
	MatchID string         `json:"matchId,omitempty"`
	User    *profiles.Card `json:"user,omitempty"`
}

// SwipeOutcome is the repository-level result of a swipe.
type SwipeOutcome struct {
	Action     string
	Matched    bool // a match exists after this call and it was created by this call
	MatchID    string
	NewMatch   bool
	AlreadyDid bool
}

type MatchRow struct {
	ID          string
	OtherID     string
	CreatedAt   time.Time
	SortAt      time.Time
	LastBody    *string
	LastSender  *string
	LastAt      *time.Time
	UnreadCount int
}
