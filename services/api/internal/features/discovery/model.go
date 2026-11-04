package discovery

import "example.com/api/internal/features/profiles"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultDeckSize = 10
	MaxDeckSize     = 20
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required"`
	Action string `json:"action" binding:"required"`
}

// MatchInfo describes a freshly created (or existing) match.
type MatchInfo struct {
	MatchID        string                  `json:"matchId"`
	ConversationID string                  `json:"conversationId"`
	Profile        *profiles.PublicProfile `json:"profile,omitempty"`
}

type SwipeResponse struct {
	// Result is "liked", "passed" or "matched".
	Result string `json:"result"`
	// AlreadyProcessed is true when this exact target had been judged before;
	// the call is idempotent and changes nothing.
	AlreadyProcessed bool       `json:"alreadyProcessed"`
	Match            *MatchInfo `json:"match,omitempty"`
}

type DeckResponse struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}

type swipeOutcome struct {
	Inserted       bool
	Matched        bool
	MatchID        string
	ConversationID string
}
