package safety

import (
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	ErrNotFound   = platformerrors.New(http.StatusNotFound, "not_found", "user not found")
	ErrSelf       = platformerrors.New(http.StatusBadRequest, "invalid_request", "you cannot do this to yourself")
	ErrBadReason  = platformerrors.New(http.StatusBadRequest, "invalid_request", "unknown report reason")
	ErrLongDetail = platformerrors.New(http.StatusBadRequest, "invalid_request", "details must be at most 1000 characters")
)

var validReasons = map[string]bool{
	"spam": true, "fake_profile": true, "inappropriate_content": true,
	"harassment": true, "underage": true, "scam": true, "other": true,
}

type UserRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type ReportRequest struct {
	UserID  string `json:"userId" binding:"required,uuid"`
	Reason  string `json:"reason" binding:"required"`
	Details string `json:"details"`
}

type Block struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}
