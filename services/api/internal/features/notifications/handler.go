package notifications

import (
	"errors"
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
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		httpx.Internal(c, "notifications.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid request")
		return
	}
	n, err := h.service.Create(c.Request.Context(), httpx.UserID(c), req.Type, req.Title, req.Body, nil)
	if err != nil {
		for _, known := range []error{ErrTypeRequired, ErrTitleRequired, ErrBodyRequired, ErrTypeTooLong, ErrTitleTooLong, ErrBodyTooLong} {
			if errors.Is(err, known) {
				httpx.BadRequest(c, "invalid_notification", known.Error())
				return
			}
		}
		httpx.Internal(c, "notifications.create", err)
		return
	}
	c.JSON(http.StatusCreated, n)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := strings.TrimSpace(c.Param("notificationId"))
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "invalid_notification_id", "invalid notification id")
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.Fail(c, "notifications.mark_read", http.StatusNotFound, "notification_not_found", "notification not found", nil)
			return
		}
		httpx.Internal(c, "notifications.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	n, err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.mark_all_read", err)
		return
	}
	c.JSON(http.StatusOK, ReadAllResponse{Updated: n})
}
