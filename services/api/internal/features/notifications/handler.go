package notifications

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit := DefaultLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.BadRequest(c, "limit must be between 1 and 50")
			return
		}
		limit = n
	}
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "before must be an RFC 3339 timestamp")
			return
		}
		before = &t
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
