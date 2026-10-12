package matching

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.TargetID) {
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid target")
		return
	}
	req.TargetID = lower(req.TargetID)
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, res)
	case errors.Is(err, ErrValidation):
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid swipe")
	case errors.Is(err, ErrOnboardingIncomplete):
		httpx.Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Fail(c, http.StatusNotFound, "profile not available")
	default:
		httpx.Internal(c, "matching.swipe", err)
	}
}

func (h *Handler) List(c *gin.Context) {
	limit, offset := 0, 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.Fail(c, http.StatusBadRequest, "limit must be between 1 and 50")
			return
		}
		limit = n
	}
	if raw := c.Query("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > MaxOffset {
			httpx.Fail(c, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		httpx.Internal(c, "matching.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "match not found")
			return
		}
		httpx.Internal(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func lower(s string) string { return strings.ToLower(s) }
