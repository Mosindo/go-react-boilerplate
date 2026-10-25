package safety

import "time"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	ThumbURL  *string   `json:"thumbUrl"`
	CreatedAt time.Time `json:"createdAt"`
}
