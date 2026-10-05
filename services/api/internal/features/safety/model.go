package safety

import (
	"time"

	"example.com/api/internal/features/profiles"
)

var ReportReasons = []string{"spam", "fake_profile", "inappropriate_content", "harassment", "underage", "scam", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
}

type ReportRequest struct {
	UserID    string `json:"userId" binding:"required,max=36"`
	Reason    string `json:"reason" binding:"required,max=40"`
	Details   string `json:"details" binding:"max=4000"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type BlockedUser struct {
	User      profiles.Card `json:"user"`
	BlockedAt time.Time     `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}
