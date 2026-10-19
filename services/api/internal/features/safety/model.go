package safety

import (
	"errors"
	"time"
)

const MaxDetailsRunes = 1000

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrSelfAction    = errors.New("cannot target yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrInvalidInput  = errors.New("invalid input")
)

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	PhotoID   *string   `json:"photoId,omitempty"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

// Removed describes a conversation destroyed by a block, so the other side's
// clients can drop it in real time.
type Removed struct {
	MatchID        string
	ConversationID string
}
