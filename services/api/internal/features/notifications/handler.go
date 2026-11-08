package notifications

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "invalid 'before' cursor")
			return
		}
		before = &t
	}
	res, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("notificationId")); err != nil {
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
