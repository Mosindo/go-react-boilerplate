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
	UserID string `json:"userId"`
	Action string `json:"action"`
}

type Match struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	HasMessages    bool          `json:"hasMessages"`
	User           profiles.Card `json:"user"`
}

type SwipeResponse struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	Match   *Match `json:"match,omitempty"`
}

type MatchesResponse struct {
	Matches    []Match    `json:"matches"`
	NextBefore *time.Time `json:"nextBefore,omitempty"`
}

type storedMatch struct {
	ID             string
	ConversationID string
	OtherUserID    string
	CreatedAt      time.Time
	HasMessages    bool
}
