package chat

import (
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

// conversationID validates the path parameter; malformed ids are plain "not found".
func conversationID(c *gin.Context, operation string) (string, bool) {
	id := strings.ToLower(c.Param("conversationId"))
	if !httpx.IsUUID(id) {
		httpx.Fail(c, operation, ErrConversationNotFound)
		return "", false
	}
	return id, true
}

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Page(c, defaultChatsLimit, maxChatsLimit)
	if !ok {
		return
	}
	list, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := conversationID(c, "chat.messages")
	if !ok {
		return
	}
	limit, _, ok := httpx.Page(c, defaultMessagesLimit, maxMessagesLimit)
	if !ok {
		return
	}
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, "chat.messages", ErrInvalidCursor)
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, before, limit)
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := conversationID(c, "chat.send")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.Bind(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := conversationID(c, "chat.read")
	if !ok {
		return
	}
	readAt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{ReadAt: readAt})
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := conversationID(c, "chat.hide")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
