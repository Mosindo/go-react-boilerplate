package matching

import "time"

const (
	DecisionLike = "like"
	DecisionPass = "pass"
)

type SwipeRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required,len=36"`
	Decision     string `json:"decision" binding:"required,oneof=like pass"`
}

// UserSummary is the compact representation of a match.
type UserSummary struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
}

type Match struct {
	ID          string      `json:"id"`
	User        UserSummary `json:"user"`
	MatchedAt   time.Time   `json:"matchedAt"`
	HasMessages bool        `json:"hasMessages"`
}

type SwipeResponse struct {
	Decision string `json:"decision"`
	Matched  bool   `json:"matched"`
	Match    *Match `json:"match,omitempty"`
}

type MatchesResponse struct {
	Matches []Match `json:"matches"`
}

type storedSummary struct {
	UserID      string
	FirstName   string
	Birthdate   time.Time
	PhotoID     *string
	MatchID     string
	MatchedAt   time.Time
	HasMessages bool
}

type swipeResult struct {
	Matched   bool
	MatchID   string
	MatchedAt time.Time
}
