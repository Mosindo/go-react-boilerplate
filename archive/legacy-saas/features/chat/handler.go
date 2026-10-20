package chat

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

func (h *Handler) Chats(c *gin.Context) {
	userID := c.GetString("userID")
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "chat.list.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	limit := parseBoundedLimit(c.Query("limit"), 20, 100)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	chats, err := h.service.ListChatsWithPagination(c.Request.Context(), organizationID, userID, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "chat.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch chats"})
		return
	}

	payload := make([]ChatSummaryResponse, 0, len(chats))
	for _, chat := range chats {
		item := ChatSummaryResponse{
			User: UserResponse{
				ID:        chat.UserID,
				Email:     chat.UserEmail,
				CreatedAt: chat.UserCreatedAt,
			},
		}
		if chat.LastMessage != nil {
			item.LastMessage = &ChatMessagePreviewResponse{
				Content:   chat.LastMessage.Content,
				CreatedAt: chat.LastMessage.CreatedAt,
			}
		}
		payload = append(payload, item)
	}

	c.JSON(http.StatusOK, ChatsResponse{Chats: payload})
}

func (h *Handler) ChatMessages(c *gin.Context) {
	userID := c.GetString("userID")
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "chat.messages.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	otherUserID := strings.TrimSpace(c.Param("userId"))
	if !isUUIDLike(otherUserID) {
		logger.LogHandlerError(c, "chat.messages.validate_user_id", http.StatusBadRequest, errors.New("invalid user id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if otherUserID == userID {
		logger.LogHandlerError(c, "chat.messages.validate_self", http.StatusBadRequest, errors.New("cannot open self chat"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot open self chat"})
		return
	}

	limit := parsePositiveInt(c.Query("limit"), 200)
	messages, err := h.service.ListMessagesWithLimit(c.Request.Context(), organizationID, userID, otherUserID, limit)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrValidateChatTarget):
			c.JSON(status, gin.H{"error": "could not validate chat target"})
		case errors.Is(err, ErrUserNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "user not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch messages"})
		}
		logger.LogHandlerError(c, "chat.messages", status, err)
		return
	}

	payload := make([]ChatMessageResponse, 0, len(messages))
	for _, message := range messages {
		payload = append(payload, ChatMessageResponse{
			ID:              message.ID,
			SenderUserID:    message.SenderUserID,
			RecipientUserID: message.RecipientUserID,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, ChatMessagesResponse{Messages: payload})
}

func (h *Handler) SendChatMessage(c *gin.Context) {
	userID := c.GetString("userID")
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "chat.send.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	otherUserID := strings.TrimSpace(c.Param("userId"))
	if !isUUIDLike(otherUserID) {
		logger.LogHandlerError(c, "chat.send.validate_user_id", http.StatusBadRequest, errors.New("invalid user id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if otherUserID == userID {
		logger.LogHandlerError(c, "chat.send.validate_self", http.StatusBadRequest, errors.New("cannot message yourself"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot message yourself"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "chat.send.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	message, err := h.service.SendMessage(c.Request.Context(), organizationID, userID, otherUserID, req.Content)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrValidateChatTarget):
			c.JSON(status, gin.H{"error": "could not validate chat target"})
		case errors.Is(err, ErrUserNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "user not found"})
		case errors.Is(err, ErrMessageContentNeeded):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "message content required"})
		default:
			c.JSON(status, gin.H{"error": "could not send message"})
		}
		logger.LogHandlerError(c, "chat.send", status, err)
		return
	}

	status := http.StatusCreated
	c.JSON(status, ChatMessageResponse{
		ID:              message.ID,
		SenderUserID:    message.SenderUserID,
		RecipientUserID: message.RecipientUserID,
		Content:         message.Content,
		CreatedAt:       message.CreatedAt,
	})
	logger.LogHandlerEvent(c, "chat.send.success", status, map[string]string{
		"recipient_user_id": otherUserID,
		"message_id":        message.ID,
	})
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
