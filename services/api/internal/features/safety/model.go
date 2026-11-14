package safety

import "time"

// SuspensionThreshold: once this many distinct members have open reports
// against someone, the account is hidden from discovery pending review.
const SuspensionThreshold = 3

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}
