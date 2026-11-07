package discovery

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusForbidden, "profile_incomplete", "complete your profile (details, preferences and at least one photo) first")
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrAlreadyDecided):
		httpx.Fail(c, http.StatusConflict, "already_decided", "you already reacted to this profile")
	default:
		logger.LogHandlerError(c, op, 500, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Feed(c *gin.Context) {
	limit := DefaultBatch
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxBatch {
			httpx.BadRequest(c, "invalid limit")
			return
		}
		limit = v
	}
	cards, err := h.service.Feed(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "discovery.feed", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId and action (like|pass) are required")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 20, 50)
	if !ok {
		return
	}
	cards, err := h.service.LikesReceived(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "discovery.likes_received", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Matches(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 30, 100)
	if !ok {
		return
	}
	list, err := h.service.Matches(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "discovery.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"matches": list})
}

func (h *Handler) Unmatch(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
