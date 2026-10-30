package safety

import "time"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "inappropriate_content": {}, "harassment": {}, "underage": {}, "other": {},
}

const (
	maxReportDetails      = 1000
	maxReportsPerDay      = 20
	maxBlocksPerUserLimit = 500
)

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=4000"`
	// Block also blocks the reported user in the same step.
	Block bool `json:"block"`
}

type BlockedUser struct {
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName"`
	BlockedAt   time.Time `json:"blockedAt"`
}

type BlockedUsersResponse struct {
	Blocked []BlockedUser `json:"blocked"`
}
