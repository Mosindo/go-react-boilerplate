package safety

import (
	"time"

	"example.com/api/internal/features/profiles"
)

var reportReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=1000"`
}

type BlockedUser struct {
	BlockedAt time.Time              `json:"blockedAt"`
	Profile   profiles.PublicProfile `json:"profile"`
}
