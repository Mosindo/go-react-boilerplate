package notifications

import (
	"errors"
	"net/http"

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
	limit, ok := httpx.Limit(c, defaultLimit, maxLimit)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	var list []Notification
	var err error
	if cursor != nil {
		list, err = h.service.List(c.Request.Context(), httpx.UserID(c), &cursor.At, cursor.ID, limit)
	} else {
		list, err = h.service.List(c.Request.Context(), httpx.UserID(c), nil, "", limit)
	}
	if err != nil {
		httpx.Internal(c, err)
		return
	}

	resp := NotificationsResponse{Notifications: make([]NotificationResponse, 0, len(list))}
	for _, n := range list {
		resp.Notifications = append(resp.Notifications, ToResponse(n))
	}
	if len(list) == limit {
		last := list[len(list)-1]
		resp.NextCursor = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	count, err := h.service.UnreadCount(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, UnreadCountResponse{Unread: count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	notificationID, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), notificationID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			httpx.NotFound(c, "Notification introuvable.")
			return
		}
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, ToResponse(n))
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
