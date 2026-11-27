package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

const (
	MaxReportDetailsRunes = 1000
	// A profile reported by this many distinct people within the window is
	// hidden from discovery until a moderator reviews it.
	AutoHideReporterThreshold = 3
	AutoHideWindow            = 30 * 24 * time.Hour
	EvidenceMessageLimit      = 50
)

type BlockedUser struct {
	UserID    string
	FirstName string
	BlockedAt time.Time
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
	// Block defaults to true: reporting someone also blocks them.
	Block *bool `json:"block"`
}

type BlockedUserResponse struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUserResponse `json:"blocks"`
}

type ReportResponse struct {
	Status  string `json:"status"`
	Blocked bool   `json:"blocked"`
}
