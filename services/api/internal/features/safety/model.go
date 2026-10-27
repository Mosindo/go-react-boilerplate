package safety

import "time"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

const maxDetailsRunes = 1000

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
	// Block additionally blocks the reported user in the same action.
	Block bool `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
