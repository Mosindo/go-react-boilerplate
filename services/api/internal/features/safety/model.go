package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

const (
	maxReportDetails = 1000
	// Accounts reported by this many distinct members within reviewWindow are
	// hidden from discovery until a moderator reviews them.
	reviewThreshold = 3
	reviewWindow    = 30 * 24 * time.Hour
)

type BlockedUser struct {
	UserID    string
	FirstName string
	BlockedAt time.Time
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=2000"`
}

type BlockedUserResponse struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type RemovedMatch struct {
	MatchID        string
	ConversationID string
}
