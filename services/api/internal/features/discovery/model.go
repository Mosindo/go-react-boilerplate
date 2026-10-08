package discovery

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,max=40"`
	Action string `json:"action" binding:"required,oneof=like pass"`
}

type SwipeResult struct {
	Action         string `json:"action"`
	Matched        bool   `json:"matched"`
	AlreadyDone    bool   `json:"alreadyProcessed"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type MatchItem struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	HasMessages    bool                   `json:"hasMessages"`
	User           profiles.PublicProfile `json:"user"`
}

type MatchesResponse struct {
	Matches    []MatchItem `json:"matches"`
	NextCursor string      `json:"nextCursor,omitempty"`
}
