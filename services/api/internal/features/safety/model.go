package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

const (
	MaxReportDetailsRunes = 1000
	// AutoHideReportThreshold hides a profile from discovery, pending manual
	// review, once this many distinct members have reported it.
	AutoHideReportThreshold = 3
	// UnderageHideThreshold is lower: two distinct "underage" reports hide
	// the profile pending review. A single report cannot hide anyone.
	UnderageHideThreshold = 2
)

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
