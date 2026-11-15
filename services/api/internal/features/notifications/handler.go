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
	cursor, err := httpx.DecodeCursor(c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "notifications.mark_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Badges(c *gin.Context) {
	b, err := h.service.Badges(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "notifications.badges", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, b)
}
