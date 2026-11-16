package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResponse struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	Match   *Match `json:"match,omitempty"`
}

type LastMessage struct {
	Body      string    `json:"body"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type Match struct {
	ID            string        `json:"id"`
	CreatedAt     time.Time     `json:"createdAt"`
	LastMessageAt *time.Time    `json:"lastMessageAt"`
	User          profiles.Card `json:"user"`
	LastMessage   *LastMessage  `json:"lastMessage"`
	UnreadCount   int           `json:"unreadCount"`
}

type ListResponse struct {
	Matches []Match `json:"matches"`
	HasMore bool    `json:"hasMore"`
}

// swipeOutcome is what the repository reports after recording a swipe.
type swipeOutcome struct {
	Action       string
	MatchID      string
	MatchCreated bool
}

type storedMatch struct {
	ID            string
	OtherUserID   string
	CreatedAt     time.Time
	LastMessageAt *time.Time
	LastMessage   *LastMessage
	UnreadCount   int
}
