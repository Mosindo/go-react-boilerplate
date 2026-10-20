package safety

import "example.com/api/internal/features/profiles"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "inappropriate_content": {}, "harassment": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,max=36"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,max=36"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=4000"`
}

type BlocksResult struct {
	Blocks []profiles.UserSummary `json:"blocks"`
}

// removedMatch describes a match torn down by a block, so clients can be told.
type removedMatch struct {
	MatchID        string
	ConversationID string
}
