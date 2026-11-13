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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrMessageContentNeeded), errors.Is(err, ErrMessageTooLong), errors.Is(err, ErrSelfConversation):
		httpx.FailCode(c, operation, http.StatusBadRequest, "validation", err.Error(), err)
	case errors.Is(err, ErrConversationNotFound):
		httpx.Fail(c, operation, http.StatusNotFound, "conversation not found", err)
	default:
		httpx.Fail(c, operation, http.StatusInternalServerError, "could not process conversation", err)
	}
}

func (h *Handler) Chats(c *gin.Context) {
	chats, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), defaultChatsLimit, maxChatsLimit), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ChatsResponse{Chats: chats})
}

func (h *Handler) Messages(c *gin.Context) {
	other, ok := httpx.UUIDParam(c, "userId", "chat.messages.validate")
	if !ok {
		return
	}
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.BadRequest(c, "chat.messages.cursor", "invalid cursor", nil)
		return
	}
	page, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), other, before,
		httpx.Limit(c.Query("limit"), defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	other, ok := httpx.UUIDParam(c, "userId", "chat.send.validate")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "chat.send.bind", "message content required", err)
		return
	}
	msg, err := h.service.SendMessage(c.Request.Context(), httpx.UserID(c), other, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	logger.LogHandlerEvent(c, "chat.send.success", http.StatusCreated, map[string]string{"message_id": msg.ID})
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	other, ok := httpx.UUIDParam(c, "userId", "chat.read.validate")
	if !ok {
		return
	}
	readAt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), other)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"readAt": readAt})
}

func (h *Handler) Clear(c *gin.Context) {
	other, ok := httpx.UUIDParam(c, "userId", "chat.clear.validate")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), other); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
