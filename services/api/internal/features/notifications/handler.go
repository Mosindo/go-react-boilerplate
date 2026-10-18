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
	limit, ok := httpx.ParseLimit(c.Query("limit"), 20, 50)
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "limit must be between 1 and 50")
		return
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), c.Query("cursor"), limit)
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			httpx.Error(c, http.StatusBadRequest, "invalid cursor")
			return
		}
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not fetch notifications")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("notificationId")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not update notification")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not update notifications")
		return
	}
	c.Status(http.StatusNoContent)
}
