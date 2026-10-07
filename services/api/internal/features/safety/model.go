package safety

import "strings"

var validReasons = map[string]struct{}{
	"spam": {}, "fake_profile": {}, "harassment": {}, "inappropriate_content": {}, "underage": {}, "other": {},
}

type BlockRequest struct {
	UserID string `json:"userId" binding:"required"`
}

type ReportRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
	Reason       string `json:"reason" binding:"required"`
	Details      string `json:"details"`
}

func normalizeReason(r string) (string, bool) {
	r = strings.ToLower(strings.TrimSpace(r))
	_, ok := validReasons[r]
	return r, ok
}
