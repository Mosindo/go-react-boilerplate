package notifications

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, msg = http.StatusNotFound, "notification not found"
	case errors.Is(err, ErrInvalidCursor):
		status, msg = http.StatusBadRequest, "invalid cursor"
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, gin.H{"error": msg})
}

func (h *Handler) List(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = v
	}
	out, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, out)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "notifications.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		h.fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
