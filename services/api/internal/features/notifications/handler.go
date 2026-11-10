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
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.Limit(c, DefaultLimit, MaxLimit))
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			httpx.BadRequest(c, "invalid_cursor", err.Error())
			return
		}
		httpx.Internal(c, "notifications.list", err)
		return
	}
	out := make([]NotificationResponse, 0, len(page.Items))
	for _, n := range page.Items {
		out = append(out, ToResponse(n))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, ListResponse{Notifications: out, UnreadCount: page.UnreadCount, NextCursor: page.NextCursor})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread_count", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
			return
		}
		httpx.Internal(c, "notifications.mark_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
