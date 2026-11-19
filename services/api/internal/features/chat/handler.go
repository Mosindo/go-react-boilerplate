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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func cursorParam(c *gin.Context) (string, bool) {
	cursor := strings.ToLower(strings.TrimSpace(c.Query("cursor")))
	if cursor != "" && !httpx.IsUUID(cursor) {
		httpx.BadRequest(c, "cursor must be a valid id")
		return "", false
	}
	return cursor, true
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.Limit(c, 20, 50)
	if !ok {
		return
	}
	cursor, ok := cursorParam(c)
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), cursor, limit)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	res, err := h.service.Get(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.get", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	limit, ok := httpx.Limit(c, 30, 100)
	if !ok {
		return
	}
	cursor, ok := cursorParam(c)
	if !ok {
		return
	}
	res, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, cursor, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, ErrInvalidMessage.Error())
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
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
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		// Non-members get the same answer as for a missing conversation.
		httpx.Fail(c, http.StatusNotFound, "conversation_not_found", "conversation not found")
	case errors.Is(err, ErrClosed):
		httpx.Fail(c, http.StatusForbidden, "conversation_closed", err.Error())
	case errors.Is(err, ErrInvalidMessage), errors.Is(err, ErrInvalidCursor):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
