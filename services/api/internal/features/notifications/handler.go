package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, err := httpx.Limit(c)
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	items, next, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	var cursor *string
	if next != "" {
		cursor = &next
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items, "nextCursor": cursor})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.read_all", err)
		return
	}
	httpx.NoContent(c)
}
