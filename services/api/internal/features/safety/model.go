package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "scam", "underage", "other"}

const (
	MaxDetailsRunes = 1000
	// AutoReviewThreshold: distinct reporters (with a real profile) after which
	// a profile is hidden from discovery pending human review.
	AutoReviewThreshold = 3
)

type BlockedUser struct {
	UserID    string
	FirstName string
	CreatedAt time.Time
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUserResponse struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
