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

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, httpx.DefaultLimit)
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkRead(c *gin.Context) {
	err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	default:
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
