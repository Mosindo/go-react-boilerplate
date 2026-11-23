package safety

import "time"

const (
	// SuspensionThreshold is the number of distinct people with open reports
	// against an account that hides it from discovery pending human review.
	SuspensionThreshold = 3
	MaxReportDetails    = 1000
)

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "scam", "underage", "other"}

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

type ReportResponse struct {
	ID      string `json:"id"`
	Blocked bool   `json:"blocked"`
}
