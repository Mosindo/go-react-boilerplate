package moderation

import "time"

// SuspensionThreshold is the number of distinct reporters that triggers an
// automatic suspension pending human review.
const SuspensionThreshold = 5

const (
	maxDetailsRunes = 1000
)

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
	MatchID string `json:"matchId"`
	Block   bool   `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}
