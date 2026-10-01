package notifications

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	before, ok, err := httpx.Cursor(c, "before")
	if err != nil {
		httpx.BadRequest(c, "curseur invalide")
		return
	}
	var cursor *time.Time
	if ok {
		cursor = &before
	}
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 100), cursor)
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	unread, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items, "unreadCount": unread})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	ok, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Internal(c, "notifications.read", err)
		return
	}
	if !ok {
		httpx.Error(c, http.StatusNotFound, "not_found", "notification introuvable")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
