package safety

import "time"

var ReportReasons = []string{"spam", "fake", "harassment", "inappropriate", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID    string `json:"userId"`
	Reason    string `json:"reason"`
	Details   string `json:"details"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
