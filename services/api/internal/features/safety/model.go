package safety

import (
	"time"

	"example.com/api/internal/features/profiles"
)

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=64"`
}

type BlockItem struct {
	User      profiles.ProfileSummary `json:"user"`
	CreatedAt time.Time               `json:"createdAt"`
}

type BlocksResponse struct {
	Blocks []BlockItem `json:"blocks"`
}

type Reason string

var validReasons = map[Reason]struct{}{
	"spam": {}, "fake": {}, "harassment": {}, "inappropriate_photos": {}, "underage": {}, "other": {},
}

func (r Reason) Valid() bool { _, ok := validReasons[r]; return ok }

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=64"`
	Reason  Reason `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type ReportResponse struct {
	ID string `json:"id"`
}

const (
	maxDetailsRunes  = 1000
	maxReportsPerDay = 20
)

// DissolvedMatch is a match removed because of a block (for realtime fan-out).
type DissolvedMatch struct {
	MatchID        string
	ConversationID string
	UserIDs        [2]string
}
