package safety

import "time"

var Reasons = map[string]struct{}{
	"spam": {}, "fake": {}, "harassment": {}, "inappropriate": {}, "underage": {}, "other": {},
}

const maxDetailsRunes = 1000

type BlockRequest struct {
	UserID string `json:"userId"`
}

type ReportRequest struct {
	UserID  string `json:"userId"`
	Reason  string `json:"reason"`
	Details string `json:"details"`
	Block   bool   `json:"block"`
}

type BlockedUser struct {
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName"`
	BlockedAt   time.Time `json:"blockedAt"`
}

// RemovedMatch identifies a match closed by a block, for realtime fan-out.
type RemovedMatch struct {
	MatchID        string
	ConversationID string
	UserIDs        [2]string
}
