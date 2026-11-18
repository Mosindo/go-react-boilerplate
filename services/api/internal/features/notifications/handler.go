package notifications

import (
	"errors"
	"net/http"
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

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), defaultLimit, maxLimit)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	items, unread, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not fetch notifications", "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, NotificationsResponse{Notifications: items, UnreadCount: unread})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not count notifications", "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, UnreadCountResponse{UnreadCount: count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if !httpx.IsUUID(notificationID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid notification id", "notifications.read.validate", nil)
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), notificationID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "notification not found", "notifications.read", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not update notification", "notifications.read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not update notifications", "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
