package safety

import "time"

const (
	// ReviewThreshold is the number of distinct reporters (over 30 days) after which an
	// account is hidden from discovery and queued for human review.
	ReviewThreshold = 3
	maxDetailsRunes = 1000
)

var validReasons = map[string]struct{}{
	"fake_profile": {}, "inappropriate_content": {}, "harassment": {}, "spam": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
