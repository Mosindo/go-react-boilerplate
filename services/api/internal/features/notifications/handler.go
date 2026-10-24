package notifications

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 20, 50)
	if !ok {
		return
	}
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"), c.Query("unread") == "true", limit, offset)
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"notifications": items})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unread": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, "notifications.read", http.StatusNotFound, "notification not found", err)
			return
		}
		httpx.Internal(c, "notifications.read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Internal(c, "notifications.readall", err)
		return
	}
	c.Status(http.StatusNoContent)
}
