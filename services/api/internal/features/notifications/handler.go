package notifications

import (
	"errors"
	"net/http"

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
	limit, ok := httpx.QueryLimit(c, 20, 50)
	if !ok {
		return
	}
	offset, ok := httpx.QueryOffset(c)
	if !ok {
		return
	}
	list, unread, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.FailInternal(c, "notifications.list", err)
		return
	}
	payload := make([]NotificationResponse, 0, len(list))
	for _, n := range list {
		payload = append(payload, toResponse(n))
	}
	c.JSON(http.StatusOK, ListResponse{Notifications: payload, UnreadCount: unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		httpx.FailInternal(c, "notifications.mark_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.FailInternal(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
