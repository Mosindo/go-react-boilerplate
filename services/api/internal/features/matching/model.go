package matching

import "example.com/api/internal/features/profiles"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoverLimit = 10
	MaxDiscoverLimit     = 20
)

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Action       string `json:"action" binding:"required"`
}

type MatchInfo struct {
	MatchID        string                 `json:"matchId"`
	ConversationID string                 `json:"conversationId"`
	User           profiles.PublicProfile `json:"user"`
}

type SwipeResponse struct {
	Action    string     `json:"action"`
	Matched   bool       `json:"matched"`
	Duplicate bool       `json:"duplicate"`
	Match     *MatchInfo `json:"match,omitempty"`
}

// viewer holds what the server knows about the person asking for candidates.
type viewer struct {
	Gender      string
	Age         int
	Preferences profiles.Preferences
	HasPhoto    bool
	HasLocation bool
}

type swipeResult struct {
	Action         string
	Duplicate      bool
	MatchID        string
	ConversationID string
	NewMatch       bool
}
