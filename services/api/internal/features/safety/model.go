package safety

import (
	"errors"
	"time"
)

var (
	ErrBadRequest = errors.New("invalid request")
	ErrSelf       = errors.New("cannot target yourself")
	ErrNotFound   = errors.New("user not found")
)

const maxDetailsLen = 1000

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {},
	"underage": {}, "scam": {}, "other": {},
}

type userIDRequest struct {
	UserID string `json:"userId"`
}

type reportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
	Block   bool   `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

// removedMatch describes a match deleted as a side effect of a block.
type removedMatch struct {
	UserA, UserB   string
	ConversationID string
}
