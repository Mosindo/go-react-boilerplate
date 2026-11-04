package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) List(c *gin.Context) {
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.Limit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
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
		httpx.Fail(c, "notifications.readall", err)
		return
	}
	c.Status(http.StatusNoContent)
}
