package safety

import "time"

var validReasons = map[string]struct{}{
	"fake_profile": {}, "inappropriate_content": {}, "harassment": {}, "spam": {}, "underage": {}, "other": {},
}

const maxDetailsRunes = 1000

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
	// Block defaults to true: reporting someone also removes them from your view.
	Block *bool `json:"block"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

type Report struct {
	ID             string     `json:"id"`
	ReporterID     *string    `json:"reporterId"`
	ReportedUserID string     `json:"reportedUserId"`
	ReportedName   string     `json:"reportedName"`
	Reason         string     `json:"reason"`
	Details        string     `json:"details"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
}

type ReportsResponse struct {
	Reports []Report `json:"reports"`
}

type ResolveReportRequest struct {
	Status      string `json:"status" binding:"required,oneof=reviewed dismissed"`
	SuspendUser bool   `json:"suspendUser"`
}
