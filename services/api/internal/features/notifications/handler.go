package notifications

import (
	"errors"
	"net/http"
	"strconv"

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
	limit := httpx.ParseLimit(c.Query("limit"), 20, 50)
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 || offset > 10_000 {
		httpx.BadRequest(c, "invalid_offset", "offset must be between 0 and 10000")
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.unread_count", err)
		return
	}
	c.JSON(http.StatusOK, UnreadCountResponse{UnreadCount: count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "notification not found", "notifications.read", nil)
			return
		}
		httpx.Internal(c, "notifications.read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
