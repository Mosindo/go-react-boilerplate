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

func (h *Handler) Chats(c *gin.Context) {
	res, err := h.service.ListChats(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func otherParam(c *gin.Context) (string, bool) {
	other := strings.ToLower(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(other) {
		httpx.BadRequest(c, "invalid_user_id", "invalid user id")
		return "", false
	}
	if other == httpx.UserID(c) {
		httpx.BadRequest(c, "self_chat", "cannot open a chat with yourself")
		return "", false
	}
	return other, true
}

func (h *Handler) Messages(c *gin.Context) {
	other, ok := otherParam(c)
	if !ok {
		return
	}
	before := strings.TrimSpace(c.Query("before"))
	if before != "" && !httpx.IsUUID(before) {
		httpx.BadRequest(c, "invalid_cursor", "invalid cursor")
		return
	}
	res, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), other, before,
		httpx.Limit(c.Query("limit"), defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	other, ok := otherParam(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "message content required")
		return
	}
	m, err := h.service.Send(c.Request.Context(), httpx.UserID(c), other, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) MarkRead(c *gin.Context) {
	other, ok := otherParam(c)
	if !ok {
		return
	}
	evt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), other)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, evt)
}

func (h *Handler) Clear(c *gin.Context) {
	other, ok := otherParam(c)
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), other); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotMatched):
		// 404 rather than 403: do not reveal whether the member exists.
		httpx.Fail(c, op, http.StatusNotFound, "not_matched", err.Error(), nil)
	case errors.Is(err, ErrMessageContentNeeded):
		httpx.Fail(c, op, http.StatusBadRequest, "content_required", err.Error(), nil)
	case errors.Is(err, ErrMessageTooLong):
		httpx.Fail(c, op, http.StatusBadRequest, "message_too_long", err.Error(), nil)
	case errors.Is(err, ErrInvalidCursor):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_cursor", err.Error(), nil)
	default:
		httpx.Internal(c, op, err)
	}
}
