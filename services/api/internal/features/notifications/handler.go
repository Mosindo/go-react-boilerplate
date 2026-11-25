package notifications

import (
	"net/http"
	"strings"

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
	before := strings.TrimSpace(c.Query("before"))
	if before != "" && !httpx.IsUUID(before) {
		c.AbortWithStatusJSON(http.StatusBadRequest, httpx.ErrorBody{Error: "invalid cursor", Code: "invalid_cursor"})
		return
	}
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "notifications.unread_count", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, UnreadCountResponse{UnreadCount: n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "notifications.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
