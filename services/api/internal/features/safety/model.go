package safety

import (
	"time"

	"example.com/api/internal/platform/storage"
)

var ReportReasons = []string{"fake_profile", "harassment", "inappropriate_content", "underage", "spam", "scam", "other"}

type ReportRequest struct {
	UserID    string `json:"userId"`
	Reason    string `json:"reason"`
	Details   string `json:"details"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type Report struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type BlockedUser struct {
	User      storage.UserSummary `json:"user"`
	BlockedAt time.Time           `json:"blockedAt"`
}
