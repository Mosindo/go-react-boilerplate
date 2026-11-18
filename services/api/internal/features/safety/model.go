package safety

import "time"

var ReportReasons = []string{"fake_profile", "inappropriate_content", "harassment", "spam", "underage", "other"}

// ModerationThreshold is the number of distinct members with an open report
// after which a profile is hidden from discovery until a moderator reviews it.
const ModerationThreshold = 3

// ModerationMinAccountAge: only reporters whose account is older than this
// count towards the threshold, so freshly created accounts cannot gang up
// to hide someone. Their reports are still stored for moderators.
const ModerationMinAccountAge = "24 hours"

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,len=36"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,len=36"`
	Reason  string `json:"reason" binding:"required,max=40"`
	Details string `json:"details" binding:"max=4000"`
}

type BlockedUser struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	PhotoURL  *string   `json:"photoUrl"`
	BlockedAt time.Time `json:"blockedAt"`
}

type BlocksResponse struct {
	Blocks []BlockedUser `json:"blocks"`
}

type storedBlock struct {
	UserID    string
	FirstName *string
	PhotoID   *string
	BlockedAt time.Time
}
