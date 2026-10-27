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
)

type SwipeRequest struct {
	TargetID string `json:"targetId"`
	Action   string `json:"action"`
}

type SwipeResult struct {
	Matched        bool   `json:"matched"`
	MatchID        string `json:"matchId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

// Match is a mutual like, as seen by one of its two members.
type Match struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	User           profiles.PublicProfile `json:"user"`
}

type MatchPage struct {
	Matches    []Match `json:"matches"`
	NextCursor string  `json:"nextCursor,omitempty"`
}
