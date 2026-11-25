package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=4000"`
	// Block defaults to true: reporting someone also blocks them.
	Block *bool `json:"block"`
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
