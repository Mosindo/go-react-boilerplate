package matching

import (
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
)

type Action string

const (
	ActionLike Action = "like"
	ActionPass Action = "pass"
)

func (a Action) Valid() bool { return a == ActionLike || a == ActionPass }

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
	Action Action `json:"action" binding:"required"`
}

type MatchRef struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
}

type SwipeResponse struct {
	Matched       bool      `json:"matched"`
	AlreadySwiped bool      `json:"alreadySwiped"`
	Match         *MatchRef `json:"match"`
}

// MatchedUser is one side of a freshly created match, with what the service
// needs to notify them in real time.
type MatchedUser struct {
	UserID       string
	Other        profiles.ProfileSummary
	Notification notifications.Notification
}

type SwipeResult struct {
	SwipeResponse
	Created []MatchedUser // non-empty only when this swipe created the match
}

type MatchItem struct {
	ID             string                  `json:"id"`
	CreatedAt      time.Time               `json:"createdAt"`
	ConversationID string                  `json:"conversationId"`
	User           profiles.ProfileSummary `json:"user"`
}

type MatchesResponse struct {
	Matches []MatchItem `json:"matches"`
}

type ProfilesResponse struct {
	Profiles []*profiles.PublicProfile `json:"profiles"`
}

// RemovedMatch identifies a match that was dissolved (for realtime fan-out).
type RemovedMatch struct {
	MatchID        string
	ConversationID string
	UserIDs        [2]string
}
