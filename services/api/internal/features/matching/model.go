package matching

import "example.com/api/internal/features/profiles"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDiscoverLimit = 10
	MaxDiscoverLimit     = 20
)

type SwipeRequest struct {
	TargetID string `json:"targetId" binding:"required,max=64"`
	Action   string `json:"action" binding:"required,max=10"`
}

type SwipeResponse struct {
	Matched bool                    `json:"matched"`
	MatchID string                  `json:"matchId,omitempty"`
	Profile *profiles.PublicProfile `json:"profile,omitempty"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}
