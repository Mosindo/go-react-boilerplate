package safety

import (
	"time"

	"example.com/api/internal/features/profiles"
)

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required,oneof=fake_profile inappropriate_content harassment spam underage other"`
	Details string `json:"details" binding:"max=4000"`
	// Block defaults to true: reporting someone also blocks them.
	Block *bool `json:"block"`
}

type BlockedUser struct {
	User      profiles.Summary `json:"user"`
	BlockedAt time.Time        `json:"blockedAt"`
}

type blockRow struct {
	UserID    string
	BlockedAt time.Time
}
