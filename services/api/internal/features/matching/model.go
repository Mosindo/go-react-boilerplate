package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
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

type Match struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	User           profiles.Card `json:"user"`
}

type LikeReceived struct {
	LikedAt time.Time     `json:"likedAt"`
	User    profiles.Card `json:"user"`
}
