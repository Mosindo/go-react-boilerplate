package notifications

import (
	"errors"
	"net/http"
	"regexp"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), defaultLimit, maxLimit)
	offset := httpx.Offset(c.Query("offset"))
	unreadOnly := c.Query("unread") == "true"
	items, unread, err := h.service.List(c.Request.Context(), c.GetString("userID"), unreadOnly, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	payload := make([]NotificationResponse, 0, len(items))
	for _, n := range items {
		payload = append(payload, ToResponse(n))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, NotificationsResponse{Notifications: payload, UnreadCount: unread})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("notificationId")
	if !uuidPattern.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "notification not found")
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "notification not found")
			return
		}
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.JSON(http.StatusOK, ToResponse(n))
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), c.GetString("userID")); err != nil {
		logger.LogHandlerError(c, "notifications.read_all", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
