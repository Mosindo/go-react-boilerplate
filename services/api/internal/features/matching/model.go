package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoverLimit = 10
	MaxDiscoverLimit     = 20
	DefaultMatchesLimit  = 30
	MaxMatchesLimit      = 50
)

// Brief is the minimal card shown for a matched user.
type Brief struct {
	UserID    string             `json:"userId"`
	FirstName string             `json:"firstName"`
	Age       int                `json:"age"`
	Photo     *profiles.PhotoRef `json:"photo,omitempty"`
}

type Match struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	User           Brief     `json:"user"`
	CreatedAt      time.Time `json:"createdAt"`
	// HasMessages lets clients separate "new matches" from ongoing conversations.
	HasMessages bool `json:"hasMessages"`
}

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
	Action string `json:"action" binding:"required,max=8"`
}

type SwipeResult struct {
	Action        string `json:"action"`
	Matched       bool   `json:"matched"`
	AlreadySwiped bool   `json:"alreadySwiped"`
	Match         *Match `json:"match,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type MatchesResponse struct {
	Matches    []Match    `json:"matches"`
	NextCursor *time.Time `json:"nextCursor,omitempty"`
}
