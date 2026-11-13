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
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), defaultLimit, maxLimit), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		httpx.Fail(c, "notifications.list", http.StatusInternalServerError, "could not fetch notifications", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId", "notifications.read.validate")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, "notifications.read", http.StatusNotFound, "notification not found", err)
			return
		}
		httpx.Fail(c, "notifications.read", http.StatusInternalServerError, "could not update notification", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.read_all", http.StatusInternalServerError, "could not update notifications", err)
		return
	}
	c.Status(http.StatusNoContent)
}
