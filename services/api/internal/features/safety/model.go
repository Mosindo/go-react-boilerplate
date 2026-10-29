package safety

import "time"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {},
	"underage": {}, "scam": {}, "other": {},
}

const maxDetails = 1000

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=64"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=5000"`
	// Block also blocks the reported user in the same call.
	Block bool `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
