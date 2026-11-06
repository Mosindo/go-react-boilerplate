package moderation

import "time"

var Reasons = []string{"spam", "fake", "harassment", "inappropriate", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=36"`
	Reason  string `json:"reason" binding:"required,max=20"`
	Details string `json:"details" binding:"max=2000"`
	Block   bool   `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}
