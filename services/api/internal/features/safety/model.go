package safety

import "time"

var Reasons = []string{"spam", "fake_profile", "harassment", "inappropriate_content", "underage", "other"}

func ValidReason(r string) bool {
	for _, v := range Reasons {
		if v == r {
			return true
		}
	}
	return false
}

const MaxDetails = 1000

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
