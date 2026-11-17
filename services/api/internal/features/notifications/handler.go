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
	limit, ok := httpx.Limit(c, defaultLimit, MaxLimit)
	if !ok {
		return
	}
	before, ok := httpx.OptionalUUIDQuery(c, "before")
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), before, limit)
	if err != nil {
		httpx.FailInternal(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.FailInternal(c, "notifications.unread_count", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, http.StatusNotFound, "notification_not_found", "Notification introuvable.")
			return
		}
		httpx.FailInternal(c, "notifications.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.FailInternal(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
