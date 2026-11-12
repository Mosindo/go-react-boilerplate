package safety

import (
	"errors"
	"time"
)

var (
	ErrSelfAction     = errors.New("you cannot block or report yourself")
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidReason  = errors.New("invalid report reason")
	ErrDetailsTooLong = errors.New("details are too long (1000 characters max)")
)

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	PhotoURL  string    `json:"photoUrl,omitempty"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

type ReportResponse struct {
	Status  string `json:"status"`
	Blocked bool   `json:"blocked"`
}

type storedBlock struct {
	UserID    string
	BlockedAt time.Time
}
