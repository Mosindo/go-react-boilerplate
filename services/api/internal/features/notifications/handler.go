package notifications

import (
	"errors"
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
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
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid limit")
			return
		}
		limit = n
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		if errors.Is(err, ErrInvalidCursor) {
			platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid cursor")
			return
		}
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not list notifications")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			platformerrors.Respond(c, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not update notification")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "could not update notifications")
		return
	}
	c.Status(http.StatusNoContent)
}
