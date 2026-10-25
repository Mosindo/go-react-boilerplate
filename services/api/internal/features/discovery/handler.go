package discovery

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/features/profiles"
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

func (h *Handler) Feed(c *gin.Context) {
	limit, err := httpx.Limit(c, DefaultFeedSize, MaxFeedSize)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	items, err := h.service.Feed(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		logger.LogHandlerError(c, "discovery.feed", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	if items == nil {
		items = []profiles.PublicProfile{}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	target := strings.ToLower(req.UserID)
	if !httpx.IsUUID(target) || (req.Action != ActionLike && req.Action != ActionPass) {
		httpx.BadRequest(c, "userId must be a UUID and action must be like or pass")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), target, req.Action)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, resp)
	case errors.Is(err, ErrTargetUnavailable):
		httpx.NotFound(c)
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusForbidden, "profile_incomplete", err.Error())
	default:
		logger.LogHandlerError(c, "discovery.swipe", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}
