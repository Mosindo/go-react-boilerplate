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
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"),
		httpx.Limit(c.Query("limit"), defaultLimit, maxLimit), httpx.Offset(c.Query("offset")))
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notifications.read", "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, "notifications.read", http.StatusNotFound, "not_found", "notification not found", err)
			return
		}
		httpx.Internal(c, "notifications.read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
