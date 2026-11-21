package chat

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
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
	limit := httpx.ParseLimit(c.Query("limit"), DefaultChatsLimit, MaxChatsLimit)
	offset := httpx.ParseOffset(c.Query("offset"))
	chats, err := h.service.ListChats(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ChatsResponse{Chats: chats})
}

func otherUser(c *gin.Context) (string, bool) {
	otherID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return "", false
	}
	return otherID, true
}

func (h *Handler) ChatMessages(c *gin.Context) {
	otherID, ok := otherUser(c)
	if !ok {
		return
	}
	var before *string
	if raw := strings.ToLower(strings.TrimSpace(c.Query("before"))); raw != "" {
		if !httpx.IsUUID(raw) {
			httpx.Error(c, http.StatusBadRequest, "invalid_cursor", "invalid before cursor")
			return
		}
		before = &raw
	}
	limit := httpx.ParseLimit(c.Query("limit"), DefaultChatMessagesLimit, MaxChatMessagesLimit)
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), otherID, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SendChatMessage(c *gin.Context) {
	otherID, ok := otherUser(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	message, err := h.service.SendMessage(c.Request.Context(), httpx.UserID(c), otherID, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

func (h *Handler) MarkRead(c *gin.Context) {
	otherID, ok := otherUser(c)
	if !ok {
		return
	}
	receipt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), otherID)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, receipt)
}

func (h *Handler) Clear(c *gin.Context) {
	otherID, ok := otherUser(c)
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), otherID); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrConversationNotFound):
		status = http.StatusNotFound
		httpx.Error(c, status, "conversation_not_found", "conversation not found")
	case errors.Is(err, ErrMessageContentNeeded), errors.Is(err, ErrMessageTooLong):
		status = http.StatusBadRequest
		httpx.Error(c, status, "invalid_message", err.Error())
	default:
		httpx.Internal(c, "could not process chat request")
	}
	logger.LogHandlerError(c, op, status, err)
}
