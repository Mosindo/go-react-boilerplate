package safety

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrSelfAction    = errors.New("you cannot do this to yourself")
	ErrInvalidReason = errors.New("invalid reason")
)

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "inappropriate_content": {},
	"harassment": {}, "underage": {}, "other": {},
}

type BlockedUser struct {
	UserID    string
	FirstName string
	BlockedAt time.Time
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=64"`
	Reason  string `json:"reason" binding:"required,max=32"`
	Details string `json:"details" binding:"max=4000"`
	// AlsoBlock blocks the reported user in the same call.
	AlsoBlock bool `json:"alsoBlock"`
}

type BlockedUserResponse struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUserResponse `json:"blocks"`
}
