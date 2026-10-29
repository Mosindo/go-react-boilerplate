package notifications

import (
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
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
	limit, ok := httpx.Limit(c, 30, 100)
	if !ok {
		return
	}
	var before *time.Time
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Fail(c, "notifications.list", platformerrors.Invalid("before must be an RFC3339 timestamp"))
			return
		}
		before = &t
	}
	resp, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		httpx.Fail(c, "notifications.read", platformerrors.NotFound("notification not found"))
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		httpx.Fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
