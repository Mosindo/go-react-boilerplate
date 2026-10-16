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
	userID := c.GetString("userID")
	limit := httpx.Limit(c.Query("limit"), 30, 100)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	list, err := h.service.List(c.Request.Context(), userID, limit, offset)
	if err != nil {
		httpx.Fail(c, "notifications.list", http.StatusInternalServerError, "could not fetch notifications", err)
		return
	}
	unread, err := h.service.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		httpx.Fail(c, "notifications.unread", http.StatusInternalServerError, "could not fetch notifications", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"notifications": list, "unread": unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		httpx.Fail(c, "notifications.read.id", http.StatusBadRequest, "invalid notification id", errors.New("invalid notification id"))
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, "notifications.read", http.StatusNotFound, "notification not found", err)
			return
		}
		httpx.Fail(c, "notifications.read", http.StatusInternalServerError, "could not update notification", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Fail(c, "notifications.read_all", http.StatusInternalServerError, "could not update notifications", err)
		return
	}
	c.Status(http.StatusNoContent)
}
