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
	before, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, httpx.QueryLimit(c, DefaultLimit, MaxLimit))
	if httpx.WriteError(c, "notifications.list", err) {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, "notifications.read", http.StatusNotFound, "notification not found", nil)
		return
	}
	if httpx.WriteError(c, "notifications.read", err) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if httpx.WriteError(c, "notifications.read_all", h.service.MarkAllRead(c.Request.Context(), c.GetString("userID"))) {
		return
	}
	c.Status(http.StatusNoContent)
}
