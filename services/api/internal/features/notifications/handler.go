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
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 100), httpx.Offset(c, 10_000))
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "notification_not_found", "notification introuvable")
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
