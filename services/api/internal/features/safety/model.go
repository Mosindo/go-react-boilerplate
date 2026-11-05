package safety

import "time"

var validReasons = map[string]bool{
	"spam": true, "fake": true, "harassment": true,
	"inappropriate": true, "underage": true, "other": true,
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID    string `json:"userId" binding:"required,uuid"`
	Reason    string `json:"reason" binding:"required"`
	Details   string `json:"details" binding:"max=1000"`
	AlsoBlock bool   `json:"alsoBlock"`
}

type BlockedUser struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

// Events notifies other modules that a match ended because of a block.
type Events interface {
	MatchEnded(matchID, conversationID, byUserID, otherUserID string)
}
