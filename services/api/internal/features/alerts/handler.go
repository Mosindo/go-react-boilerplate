package alerts

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) List(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.Fail(c, http.StatusBadRequest, "limit must be between 1 and 50")
			return
		}
		limit = n
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	page, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		httpx.Internal(c, "alerts.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "notification not found")
			return
		}
		httpx.Internal(c, "alerts.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Internal(c, "alerts.readall", err)
		return
	}
	c.Status(http.StatusNoContent)
}
