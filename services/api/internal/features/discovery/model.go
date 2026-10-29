package discovery

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultBatch = 10
	MaxBatch     = 30
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
	Action string `json:"action" binding:"required,max=8"`
}

// MatchUser is the minimal public card shown for the other person of a match.
type MatchUser struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Match struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	User           MatchUser `json:"user"`
}

type SwipeResponse struct {
	Action string `json:"action"`
	// AlreadyProcessed is true when the same swipe had already been recorded
	// (double tap, retry): nothing changed, the stored outcome is returned.
	AlreadyProcessed bool   `json:"alreadyProcessed"`
	Matched          bool   `json:"matched"`
	Match            *Match `json:"match"`
}
