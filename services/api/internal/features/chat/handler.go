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
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "conversation not found", err)
	case errors.Is(err, ErrEmptyMessage):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, "message cannot be empty", err)
	case errors.Is(err, ErrMessageTooLong):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, "message is too long (2000 characters max)", err)
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, DefaultConvLimit, MaxConvLimit)
	if !ok {
		return
	}
	items, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"conversations": items})
}

func (h *Handler) Unread(c *gin.Context) {
	n, err := h.service.UnreadTotal(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "chat.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unread": n})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	limit, _, ok := httpx.Pagination(c, DefaultMessageLimit, MaxMessageLimit)
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, "chat.messages", http.StatusBadRequest, "invalid cursor", nil)
		return
	}
	msgs, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
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
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"marked": n})
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
