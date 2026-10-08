package safety

import "time"

var validReasons = map[string]bool{
	"spam": true, "harassment": true, "fake_profile": true, "inappropriate_content": true, "underage": true, "other": true,
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=40"`
}

type ReportRequest struct {
	UserID    string `json:"userId" binding:"required,max=40"`
	Reason    string `json:"reason" binding:"required,max=40"`
	Details   string `json:"details" binding:"max=4000"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
