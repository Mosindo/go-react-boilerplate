package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("before"), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, page)
}

// MarkRead marks the listed notifications read, or every notification when no ids are sent.
func (h *Handler) MarkRead(c *gin.Context) {
	var req MarkReadRequest
	if c.Request.ContentLength != 0 && !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), req.IDs); err != nil {
		httpx.Fail(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
