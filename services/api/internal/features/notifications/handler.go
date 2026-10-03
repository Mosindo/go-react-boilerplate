package notifications

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit := defaultLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 100"})
			return
		}
		limit = n
	}
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before must be an RFC3339 timestamp"})
			return
		}
		before = &t
	}
	items, unread, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch notifications"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, NotificationsResponse{Notifications: items, UnreadCount: unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("notificationId")
	if !validate.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		logger.LogHandlerError(c, "notifications.mark_read", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update notification"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.mark_all_read", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update notifications"})
		return
	}
	c.Status(http.StatusNoContent)
}
