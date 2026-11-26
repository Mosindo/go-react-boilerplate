package notifications

import (
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
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "notifications.unread_count", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "notifications.mark_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
