package notifications

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	limit, err := httpx.Limit(c, httpx.DefaultLimit, httpx.MaxLimit)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	cur, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.NotFound(c)
			return
		}
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
