package chat

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrBlocked):
		httpx.Fail(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, err := httpx.Limit(c, httpx.DefaultLimit, httpx.MaxLimit)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	cur, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadCount(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "chat.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	limit, err := httpx.Limit(c, DefaultPageSize, httpx.MaxLimit)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	cur, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), id, limit, cur)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
