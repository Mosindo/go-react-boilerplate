package matches

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultLimit = 30
	MaxLimit     = 50
)

type LikeRequest struct {
	UserID string `json:"userId"`
	Action string `json:"action"`
}

type MatchRef struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
}

type LikeResult struct {
	Matched bool      `json:"matched"`
	Match   *MatchRef `json:"match,omitempty"`
}

// StoredMatch is a row of the matches list before profiles are attached.
type StoredMatch struct {
	ID             string
	ConversationID string
	OtherUserID    string
	CreatedAt      time.Time
	LastMessageAt  *time.Time
}

type Match struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	CreatedAt      time.Time              `json:"createdAt"`
	LastMessageAt  *time.Time             `json:"lastMessageAt"`
	Profile        profiles.PublicProfile `json:"profile"`
}

type MatchesResponse struct {
	Matches    []Match    `json:"matches"`
	NextBefore *time.Time `json:"nextBefore"`
}
