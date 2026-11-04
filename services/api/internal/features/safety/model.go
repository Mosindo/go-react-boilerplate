package safety

import "time"

var validReasons = map[string]bool{
	"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true,
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=1000"`
	// AlsoBlock blocks the reported user in the same call.
	AlsoBlock bool `json:"alsoBlock"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	PhotoURL  *string   `json:"photoUrl"`
	BlockedAt time.Time `json:"blockedAt"`
}
