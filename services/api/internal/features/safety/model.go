// Package safety implements blocking and reporting.
package safety

import "time"

var Reasons = []string{"spam", "fake", "inappropriate", "harassment", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
	// AlsoBlock blocks the reported user in the same call.
	AlsoBlock bool `json:"alsoBlock"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	ThumbURL  string    `json:"thumbUrl"`
	BlockedAt time.Time `json:"blockedAt"`
}

type StoredBlock struct {
	UserID    string
	BlockedAt time.Time
}
