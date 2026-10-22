package safety

import (
	"time"
)

var ReportReasons = []string{"spam", "fake_profile", "harassment", "inappropriate_content", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=36"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=2000"`
	// AlsoBlock blocks the reported user in the same action.
	AlsoBlock bool `json:"alsoBlock"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
