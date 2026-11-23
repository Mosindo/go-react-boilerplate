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
		// Same answer whether the conversation does not exist or is not yours.
		httpx.Fail(c, http.StatusNotFound, "not_found", "conversation not found", operation, nil)
	case errors.Is(err, ErrCursorNotFound):
		httpx.BadRequest(c, "invalid_cursor", err.Error())
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageTooLong):
		httpx.BadRequest(c, "invalid_message", err.Error())
	default:
		httpx.Internal(c, operation, err)
	}
}

func (h *Handler) ListMessages(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.BadRequest(c, "invalid_cursor", "invalid cursor")
		return
	}
	limit := httpx.ParseLimit(c.Query("limit"), DefaultMessagesPage, MaxMessagesPage)
	res, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), conversationID, before, limit)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_message", "message content is required")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), conversationID, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	receipt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), conversationID)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, receipt)
}
