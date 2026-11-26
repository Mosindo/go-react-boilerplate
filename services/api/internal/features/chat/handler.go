package chat

import (
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
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "chat.list_conversations", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadConversations(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "chat.unread_count", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) GetConversation(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	conv, err := h.service.GetConversation(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.get_conversation", err)
		return
	}
	c.JSON(http.StatusOK, conv)
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), id, cursor, httpx.Limit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "chat.list_messages", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, &req) {
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
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	receipt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, receipt)
}
