package safety

import "time"

var validReasons = map[string]bool{
	"spam": true, "fake": true, "harassment": true, "inappropriate": true, "underage": true, "other": true,
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

type BlockedUser struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlockedResponse struct {
	Blocked []BlockedUser `json:"blocked"`
}
