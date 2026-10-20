package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.Limit(c, 20, 50)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeTimeCursor(c.Query("cursor"))
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), cursor, limit)
	if err != nil {
		httpx.Respond(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Respond(c, "notifications.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Respond(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
