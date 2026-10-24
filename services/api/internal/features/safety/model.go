package safety

import (
	"errors"
	"time"
)

var (
	ErrSelfTarget = errors.New("cannot target yourself")
	ErrNotFound   = errors.New("user not found")
)

var validReasons = map[string]struct{}{
	"spam": {}, "fake": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=1000"`
	// Block also blocks the reported user in the same call.
	Block bool `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
