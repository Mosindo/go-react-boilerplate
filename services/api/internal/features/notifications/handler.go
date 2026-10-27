package notifications

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidCursor):
		httpx.Error(c, http.StatusBadRequest, "invalid cursor")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "notification not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, httpx.DefaultLimit), c.Query("cursor"))
	if err != nil {
		h.fail(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		h.fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
