package moderation

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("user not found")
	ErrSelf     = errors.New("cannot target yourself")
	ErrReason   = errors.New("invalid reason")
)

var validReasons = map[string]struct{}{
	"spam": {}, "harassment": {}, "fake": {}, "inappropriate": {}, "underage": {}, "other": {},
}

type ReportRequest struct {
	Reason  string `json:"reason" binding:"required,max=20"`
	Details string `json:"details" binding:"max=1000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}
