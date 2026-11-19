package safety

import (
	"errors"
	"time"
)

var (
	ErrSelf          = errors.New("you cannot block or report yourself")
	ErrUserNotFound  = errors.New("user not found")
	ErrInvalidReason = errors.New("reason must be one of fake_profile, inappropriate_content, harassment, spam, underage, other")
	ErrInvalidDetail = errors.New("details must be 1000 characters or fewer")
)

var Reasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type StoredBlock struct {
	UserID    string
	FirstName *string
	CreatedAt time.Time
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

// Closed describes a conversation that a block closed, so the other side can be told.
type Closed struct {
	MatchID        string
	ConversationID string
}
