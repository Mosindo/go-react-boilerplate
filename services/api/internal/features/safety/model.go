package safety

import (
	"time"

	"example.com/api/internal/features/profiles"
)

// AutoSuspendThreshold is the number of distinct members with an open report
// against a profile after which it is hidden from discovery pending review.
const AutoSuspendThreshold = 3

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRow struct {
	UserID    string
	CreatedAt time.Time
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=36"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUserResponse struct {
	User      profiles.SummaryResponse `json:"user"`
	BlockedAt time.Time                `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUserResponse `json:"blocks"`
}

type ReportResponse struct {
	ID      string `json:"id"`
	Blocked bool   `json:"blocked"`
}
