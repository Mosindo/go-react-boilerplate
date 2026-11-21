package matching

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	targetID := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if !httpx.IsUUID(targetID) {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), targetID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	status := http.StatusCreated
	c.JSON(status, resp)
	if resp.Matched && resp.Match != nil {
		logger.LogHandlerEvent(c, "matching.match", status, map[string]string{"match_id": resp.Match.ID})
	}
}

func (h *Handler) Unmatch(c *gin.Context) {
	otherID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), otherID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrSelfSwipe):
		status = http.StatusBadRequest
		httpx.Error(c, status, "invalid_swipe", err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		status = http.StatusConflict
		httpx.Error(c, status, "already_swiped", err.Error())
	case errors.Is(err, ErrProfileIncomplete):
		status = http.StatusConflict
		httpx.Error(c, status, "profile_incomplete", err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		status = http.StatusNotFound
		httpx.Error(c, status, "profile_unavailable", err.Error())
	case errors.Is(err, ErrMatchNotFound):
		status = http.StatusNotFound
		httpx.Error(c, status, "match_not_found", "match not found")
	default:
		httpx.Internal(c, "could not process request")
	}
	logger.LogHandlerError(c, op, status, err)
}
