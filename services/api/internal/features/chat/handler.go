package chat

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

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultConversationsLimit, MaxConversationsLimit)
	if !ok {
		return
	}
	before, ok := httpx.OptionalUUIDQuery(c, "before")
	if !ok {
		return
	}
	res, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), before, limit)
	if err != nil {
		httpx.FailInternal(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadConversations(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.FailInternal(c, "chat.unread_count", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) GetConversation(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	conv, err := h.service.GetConversation(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, conv)
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	limit, ok := httpx.Limit(c, DefaultMessagesLimit, MaxMessagesLimit)
	if !ok {
		return
	}
	before, ok := httpx.OptionalUUIDQuery(c, "before")
	if !ok {
		return
	}
	res, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) SendMessage(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, "chat.send", &req) {
		return
	}
	msg, err := h.service.SendMessage(c.Request.Context(), httpx.UserID(c), id, req.Content, req.ClientMessageID)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	event, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, event)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		// Same answer whether the conversation does not exist or belongs to someone else.
		httpx.Fail(c, http.StatusNotFound, "conversation_not_found", "Conversation introuvable.")
	case errors.Is(err, ErrMessageEmpty):
		httpx.Fail(c, http.StatusBadRequest, "message_empty", "Le message est vide.")
	case errors.Is(err, ErrMessageTooLong):
		httpx.Fail(c, http.StatusBadRequest, "message_too_long", "Message trop long (2000 caractères max).")
	default:
		httpx.FailInternal(c, op, err)
	}
}
