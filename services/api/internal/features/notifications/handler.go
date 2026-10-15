package notifications

import (
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
	limit, offset, ok := httpx.Page(c, 20, 50)
	if !ok {
		return
	}
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("notificationId")
	if !httpx.IsUUID(id) {
		httpx.Fail(c, "notifications.read", ErrNotificationNotFound)
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
