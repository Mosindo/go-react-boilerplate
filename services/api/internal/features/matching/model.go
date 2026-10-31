package matching

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	KindLike = "like"
	KindPass = "pass"

	DefaultFeedSize = 10
	MaxFeedSize     = 20
)

type FeedResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

// SwipeResult is returned after a like or pass.
type SwipeResult struct {
	Matched        bool           `json:"matched"`
	AlreadyHandled bool           `json:"alreadyHandled"`
	Match          *MatchResponse `json:"match,omitempty"`
}

type MatchResponse struct {
	ID             string        `json:"id"`
	ConversationID string        `json:"conversationId"`
	CreatedAt      time.Time     `json:"createdAt"`
	User           profiles.Card `json:"user"`
}

type MatchesResponse struct {
	Matches    []MatchResponse `json:"matches"`
	NextBefore *time.Time      `json:"nextBefore,omitempty"`
}

type ReceivedLikesResponse struct {
	Profiles []profiles.Card `json:"profiles"`
}

// StoredMatch is a match row from the viewer's perspective.
type StoredMatch struct {
	ID             string
	ConversationID string
	OtherUserID    string
	CreatedAt      time.Time
}

// SwipeOutcome is the repository-level result of a swipe.
type SwipeOutcome struct {
	AlreadyHandled bool
	Match          *StoredMatch
}
