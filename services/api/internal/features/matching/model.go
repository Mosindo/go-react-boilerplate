package matching

import "time"

const (
	ActionLike = "like"
	ActionPass = "pass"

	DefaultLimit = 20
	MaxLimit     = 50
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId"`
	Action       string `json:"action"`
}

type MatchUser struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	ThumbURL  *string `json:"thumbUrl"`
}

type LastMessage struct {
	Content   string    `json:"content"`
	SenderID  string    `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

type MatchSummary struct {
	ID             string       `json:"id"`
	CreatedAt      time.Time    `json:"createdAt"`
	LastActivityAt time.Time    `json:"lastActivityAt"`
	UnreadCount    int          `json:"unreadCount"`
	User           MatchUser    `json:"user"`
	LastMessage    *LastMessage `json:"lastMessage"`
}

type SwipeResponse struct {
	Action           string        `json:"action"`
	AlreadyProcessed bool          `json:"alreadyProcessed"`
	Matched          bool          `json:"matched"`
	Match            *MatchSummary `json:"match"`
}

type MatchList struct {
	Matches    []MatchSummary `json:"matches"`
	NextCursor *string        `json:"nextCursor"`
}

// SwipeOutcome is the repository result of a swipe.
type SwipeOutcome struct {
	Action           string
	AlreadyProcessed bool
	MatchID          string // empty when there is no active match
	NewMatch         bool   // true only for the swipe that created the match
}
