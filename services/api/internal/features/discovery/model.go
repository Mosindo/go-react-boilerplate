package discovery

import (
	"net/http"

	"example.com/api/internal/features/chat"
	platformerrors "example.com/api/internal/platform/errors"
)

const (
	ActionLike = "like"
	ActionPass = "pass"
)

var (
	ErrProfileRequired = platformerrors.New(http.StatusForbidden, "profile_required", "complete your profile (with at least one photo) first")
	ErrNotFound        = platformerrors.New(http.StatusNotFound, "not_found", "not found")
	ErrSelfSwipe       = platformerrors.New(http.StatusBadRequest, "invalid_request", "you cannot swipe yourself")
	ErrBadAction       = platformerrors.New(http.StatusBadRequest, "invalid_request", "action must be like or pass")
)

type SwipeRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
	Action string `json:"action" binding:"required"`
}

type SwipeResponse struct {
	Action       string             `json:"action"`
	Matched      bool               `json:"matched"`
	Conversation *chat.Conversation `json:"conversation"`
}
