package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 20, 50), c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
