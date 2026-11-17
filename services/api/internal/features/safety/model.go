package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "scam", "underage", "other"}

const maxReportDetailsRunes = 1000

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

type EndedMatch struct {
	MatchID        string
	ConversationID string
}
