package moderation

import (
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	RoleMember    = "member"
	RoleModerator = "moderator"

	StatusOpen      = "open"
	StatusReviewed  = "reviewed"
	StatusDismissed = "dismissed"
)

// Report is what moderators see. The reporter's identity is withheld; only
// the reported member is shown, with enough context to decide.
type Report struct {
	ID              string                  `json:"id"`
	Reason          string                  `json:"reason"`
	Details         string                  `json:"details"`
	Status          string                  `json:"status"`
	CreatedAt       time.Time               `json:"createdAt"`
	ReviewedAt      *time.Time              `json:"reviewedAt,omitempty"`
	ReportedUser    *profiles.Summary       `json:"reportedUser"`
	ReportedProfile *profiles.PublicProfile `json:"reportedProfile,omitempty"`
	OpenReports     int                     `json:"openReportsOnUser"`
	Suspended       bool                    `json:"reportedUserSuspended"`
}

type ReportsResponse struct {
	Reports    []Report `json:"reports"`
	NextOffset *int     `json:"nextOffset,omitempty"`
}

type ResolveRequest struct {
	Status string `json:"status" binding:"required,oneof=reviewed dismissed"`
}

type reportRow struct {
	ID          string
	Reason      string
	Details     string
	Status      string
	CreatedAt   time.Time
	ReviewedAt  *time.Time
	ReportedID  *string
	OpenReports int
	Suspended   bool
}
