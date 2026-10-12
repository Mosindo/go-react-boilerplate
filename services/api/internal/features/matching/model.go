package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

type SwipeResult struct {
	// Matched is true when this swipe, or an earlier identical one, produced a match.
	Matched bool    `json:"matched"`
	MatchID *string `json:"matchId"`
	// AlreadyProcessed is true when the viewer had already decided on this profile.
	AlreadyProcessed bool `json:"alreadyProcessed"`
}

type MessagePreview struct {
	ID        string    `json:"id"`
	SenderID  string    `json:"senderId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type MatchSummary struct {
	ID             string          `json:"id"`
	CreatedAt      time.Time       `json:"createdAt"`
	LastActivityAt time.Time       `json:"lastActivityAt"`
	Profile        profiles.Card   `json:"profile"`
	LastMessage    *MessagePreview `json:"lastMessage"`
	UnreadCount    int             `json:"unreadCount"`
}

// StoredMatch is a match row from the viewer's perspective.
type StoredMatch struct {
	ID             string
	OtherUserID    string
	CreatedAt      time.Time
	LastActivityAt time.Time
	LastMessage    *MessagePreview
	UnreadCount    int
}

type SwipeOutcome struct {
	AlreadyProcessed bool
	MatchID          string // empty when no match
}
