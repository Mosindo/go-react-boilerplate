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
	resp, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.QueryLimit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "chat.conversations", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetConversation(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	conv, err := h.service.GetConversation(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.conversation", err)
		return
	}
	c.JSON(http.StatusOK, conv)
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), id, c.Query("cursor"), httpx.QueryLimit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SendMessage(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "chat.send.bind", err)
		return
	}
	message, err := h.service.SendMessage(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
