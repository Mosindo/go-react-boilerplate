package safety

import (
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
)

// Reports from this many distinct members put a profile under review: it is
// hidden from discovery until a moderator handles the reports table.
const autoReviewThreshold = 3

var (
	ErrSelfAction     = errors.New("you cannot do this on your own account")
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidReason  = errors.New("invalid report reason")
	ErrDetailsTooLong = errors.New("details are too long")
)

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string          `json:"userId"`
	FirstName string          `json:"firstName"`
	Photo     *profiles.Photo `json:"photo,omitempty"`
	BlockedAt time.Time       `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

type ReportResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Blocked bool   `json:"blocked"`
}

type storedBlock struct {
	UserID    string
	BlockedAt time.Time
}
