package safety

import "time"

var ReportReasons = []string{"spam", "fake", "harassment", "inappropriate", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required,oneof=spam fake harassment inappropriate underage other"`
	Details string `json:"details" binding:"max=4000"`
	// Block also blocks the reported user in the same call.
	Block bool `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

type ReportResponse struct {
	ID string `json:"id"`
}
