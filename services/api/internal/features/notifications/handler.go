package notifications

import (
	"errors"
	"net/http"

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

func (h *Handler) List(c *gin.Context) {
	list, unread, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10000))
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not fetch notifications")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, NotificationsResponse{Notifications: list, UnreadCount: unread})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		logger.LogHandlerError(c, "notifications.unread", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not fetch notifications")
		return
	}
	c.JSON(http.StatusOK, UnreadCountResponse{UnreadCount: n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("notificationId")
	if !httpx.IsUUID(id) {
		httpx.Fail(c, http.StatusBadRequest, "invalid notification id")
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, http.StatusNotFound, "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.mark_read", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not update notification")
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		logger.LogHandlerError(c, "notifications.mark_all_read", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not update notifications")
		return
	}
	c.Status(http.StatusNoContent)
}
