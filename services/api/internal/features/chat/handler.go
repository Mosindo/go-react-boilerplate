package chat

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "not_found", "conversation not found", err)
	case errors.Is(err, ErrEmptyMessage):
		httpx.Fail(c, op, http.StatusBadRequest, "empty_message", err.Error(), err)
	case errors.Is(err, ErrMessageTooLong):
		httpx.Fail(c, op, http.StatusBadRequest, "message_too_long", err.Error(), err)
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"),
		httpx.Limit(c.Query("limit"), defaultConversationsLimit, maxConversationsLimit), httpx.Offset(c.Query("offset")))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "chat.messages", "conversationId")
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, "chat.messages", http.StatusBadRequest, "invalid_cursor", "invalid cursor", nil)
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), id, before,
		httpx.Limit(c.Query("limit"), defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "chat.send", "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, "chat.send", &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "chat.read", "conversationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{UnreadCount: 0})
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "chat.clear", "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
