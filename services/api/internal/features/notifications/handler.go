package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, err := httpx.ParseLimit(c, DefaultLimit, MaxLimit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
