package matching

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error(), operation, nil)
	case errors.Is(err, ErrInvalidAction):
		httpx.BadRequest(c, "invalid_action", err.Error())
	case errors.Is(err, ErrSelfSwipe):
		httpx.BadRequest(c, "self_swipe", err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", err.Error(), operation, nil)
	case errors.Is(err, ErrTargetNotAvailable):
		httpx.Fail(c, http.StatusNotFound, "not_available", err.Error(), operation, nil)
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error(), operation, nil)
	case errors.Is(err, ErrViewerNotFound):
		httpx.Fail(c, http.StatusUnauthorized, "unauthorized", "profile not found", operation, err)
	default:
		httpx.Internal(c, operation, err)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := httpx.ParseLimit(c.Query("limit"), DefaultDeckSize, MaxDeckSize)
	res, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "targetUserId and action are required")
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if !httpx.IsUUID(target) {
		httpx.BadRequest(c, "invalid_id", "invalid targetUserId")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), target, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit := httpx.ParseLimit(c.Query("limit"), 30, 50)
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 || offset > 10_000 {
		httpx.BadRequest(c, "invalid_offset", "offset must be between 0 and 10000")
		return
	}
	res, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "matching.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
