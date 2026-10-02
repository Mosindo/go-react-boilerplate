package safety

import "time"

var Reasons = []string{"spam", "fake", "inappropriate", "harassment", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=64"`
	Reason  string `json:"reason" binding:"required,max=30"`
	Details string `json:"details" binding:"max=3000"`
	Block   bool   `json:"block"`
}

type BlockedUser struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
