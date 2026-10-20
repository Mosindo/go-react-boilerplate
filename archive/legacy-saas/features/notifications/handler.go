package notifications

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("userID")
	limit := parseBoundedLimit(c.Query("limit"), 20, 100)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	notifications, err := h.service.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch notifications"})
		return
	}

	payload := make([]NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		payload = append(payload, mapNotificationResponse(notification))
	}

	c.JSON(http.StatusOK, NotificationsResponse{Notifications: payload})
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "notifications.create.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	notification, err := h.service.Create(c.Request.Context(), userID, req.Type, req.Title, req.Body)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrTypeRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification type required"})
		case errors.Is(err, ErrTitleRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification title required"})
		case errors.Is(err, ErrBodyRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification body required"})
		case errors.Is(err, ErrTypeTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification type too long"})
		case errors.Is(err, ErrTitleTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification title too long"})
		case errors.Is(err, ErrBodyTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "notification body too long"})
		default:
			c.JSON(status, gin.H{"error": "could not create notification"})
		}
		logger.LogHandlerError(c, "notifications.create", status, err)
		return
	}

	status := http.StatusCreated
	c.JSON(status, mapNotificationResponse(notification))
	logger.LogHandlerEvent(c, "notifications.create.success", status, map[string]string{
		"notification_id": notification.ID,
	})
}

func (h *Handler) MarkRead(c *gin.Context) {
	userID := c.GetString("userID")
	notificationID := strings.TrimSpace(c.Param("notificationId"))
	if !isUUIDLike(notificationID) {
		logger.LogHandlerError(c, "notifications.mark_read.validate_notification_id", http.StatusBadRequest, errors.New("invalid notification id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	notification, err := h.service.MarkRead(c.Request.Context(), userID, notificationID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrNotificationNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "notification not found"})
		default:
			c.JSON(status, gin.H{"error": "could not mark notification as read"})
		}
		logger.LogHandlerError(c, "notifications.mark_read", status, err)
		return
	}

	c.JSON(http.StatusOK, mapNotificationResponse(notification))
}

func mapNotificationResponse(notification Notification) NotificationResponse {
	return NotificationResponse{
		ID:        notification.ID,
		UserID:    notification.UserID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		IsRead:    notification.IsRead,
		CreatedAt: notification.CreatedAt,
		ReadAt:    notification.ReadAt,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseBoundedLimit(raw string, fallback, max int) int {
	parsed := parsePositiveInt(raw, fallback)
	if parsed > max {
		return max
	}
	return parsed
}

func parseNonNegativeInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func isUUIDLike(v string) bool {
	if len(v) != 36 {
		return false
	}
	hyphenPos := map[int]struct{}{8: {}, 13: {}, 18: {}, 23: {}}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if _, ok := hyphenPos[i]; ok {
			if ch != '-' {
				return false
			}
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}
