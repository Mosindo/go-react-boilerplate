package notifications

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), defaultLimit, maxLimit)
	offset := httpx.Offset(c.Query("offset"))
	list, unread, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not fetch notifications")
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, NotificationsResponse{Notifications: list, UnreadCount: unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Error(c, http.StatusNotFound, "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.mark_read", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not update notification")
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	n, err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		logger.LogHandlerError(c, "notifications.mark_all_read", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not update notifications")
		return
	}
	c.JSON(http.StatusOK, gin.H{"marked": n})
}
