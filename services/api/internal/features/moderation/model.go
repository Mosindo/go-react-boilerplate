package moderation

import (
	"errors"
	"time"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrSelf          = errors.New("cannot target yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLong   = errors.New("details too long")
)

const MaxDetailsRunes = 1000

var reasons = map[string]bool{"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID    string `json:"userId" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
	Details   string `json:"details"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type Publisher interface {
	Publish(userID string, ev realtime.Event)
}
