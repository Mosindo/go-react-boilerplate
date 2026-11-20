package chat

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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "conversation not found")
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageTooLong), errors.Is(err, ErrInvalidFilter):
		httpx.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
	default:
		httpx.FailInternal(c, operation, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.QueryLimit(c, 30, 50)
	if !ok {
		return
	}
	offset, ok := httpx.QueryOffset(c)
	if !ok {
		return
	}
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c), strings.TrimSpace(c.Query("filter")), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	payload := make([]ConversationResponse, 0, len(list))
	for _, conv := range list {
		payload = append(payload, toConversationResponse(conv))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: payload})
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "conversationId")
	if !ok {
		return
	}
	conv, err := h.service.Get(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.get", err)
		return
	}
	c.JSON(http.StatusOK, toConversationResponse(conv))
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "conversationId")
	if !ok {
		return
	}
	limit, ok := httpx.QueryLimit(c, 30, 100)
	if !ok {
		return
	}
	var before *string
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		if !httpx.IsUUID(raw) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_pagination", "before must be a message id")
			return
		}
		before = &raw
	}
	messages, hasMore, otherRead, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	payload := make([]MessageResponse, 0, len(messages))
	for _, m := range messages {
		payload = append(payload, toMessageResponse(m))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MessagesResponse{Messages: payload, HasMore: hasMore, OtherLastReadAt: otherRead})
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, "chat.send", &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, toMessageResponse(msg))
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "conversationId")
	if !ok {
		return
	}
	lastRead, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lastReadAt": lastRead})
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.PathUUID(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
