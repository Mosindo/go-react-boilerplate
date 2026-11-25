package matching

import "example.com/api/internal/features/profiles"

const (
	ActionLike = "like"
	ActionPass = "pass"
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,uuid"`
	Action       string `json:"action" binding:"required,oneof=like pass"`
}

type MatchInfo struct {
	MatchID        string                 `json:"matchId"`
	ConversationID string                 `json:"conversationId"`
	Profile        profiles.PublicProfile `json:"profile"`
}

type SwipeResponse struct {
	Action        string     `json:"action"`
	AlreadySwiped bool       `json:"alreadySwiped"`
	Match         *MatchInfo `json:"match"`
}

type DiscoverResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type swipeResult struct {
	action         string
	alreadySwiped  bool
	matchID        string
	conversationID string
	swiperName     string
	targetName     string
}
