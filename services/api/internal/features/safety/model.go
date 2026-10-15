package safety

import "time"

const (
	maxDetailsRunes = 1000
	maxBlocksLimit  = 100
)

var validReasons = map[string]struct{}{
	"spam":          {},
	"fake":          {},
	"harassment":    {},
	"inappropriate": {},
	"underage":      {},
	"other":         {},
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required,max=32"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	ThumbURL  *string   `json:"thumbUrl,omitempty"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlockedResponse struct {
	Blocked []BlockedUser `json:"blocked"`
}
