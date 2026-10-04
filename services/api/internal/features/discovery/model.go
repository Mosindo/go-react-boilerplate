package discovery

import "example.com/api/internal/features/chat"

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetID string `json:"targetId"`
	Action   string `json:"action"`
}

type SwipeResponse struct {
	Action  string             `json:"action"`
	Matched bool               `json:"matched"`
	Match   *chat.MatchSummary `json:"match"`
}

// SwipeResult is what the repository reports after a (possibly repeated) swipe.
type SwipeResult struct {
	Action        string // action stored in the database
	MatchID       string // empty when there is no active match
	NewMatch      bool   // true only for the swipe that created the match
	Notifications []NotificationOut
}

// NotificationOut is a notification created together with a match, to be pushed in realtime.
type NotificationOut struct {
	UserID string
	Value  any
}
