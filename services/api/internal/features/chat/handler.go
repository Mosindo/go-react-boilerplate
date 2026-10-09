package chat

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrBlocked):
		httpx.Forbidden(c, "you can no longer exchange messages with this person")
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, "message must contain 1 to 2000 characters")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, httpx.DefaultLimit)
	if !ok {
		return
	}
	list, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list})
}

func (h *Handler) ListMessages(c *gin.Context) {
	limit := DefaultPageSize
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxPageSize {
			httpx.BadRequest(c, "limit must be between 1 and 50")
			return
		}
		limit = v
	}
	msgs, more, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("id"), c.Query("before"), limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MessagesResponse{Messages: msgs, HasMore: more})
}

func (h *Handler) Send(c *gin.Context) {
	var req SendMessageRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), c.Param("id"), req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
