package chat

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrBlocked):
		httpx.Error(c, http.StatusForbidden, "you cannot message this user")
	case errors.Is(err, ErrInvalidBody):
		httpx.Error(c, http.StatusBadRequest, "message must be 1-2000 characters")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 50)
	cur, okCur := httpx.DecodeCursor(c)
	if !ok || !okCur {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return
	}
	items, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	next := ""
	if len(items) == limit {
		last := items[len(items)-1]
		next = httpx.EncodeCursor(last.UpdatedAt, last.ID)
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items, "nextCursor": next})
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	limit, okLimit := httpx.Limit(c, 30, 100)
	cur, okCur := httpx.DecodeCursor(c)
	if !okLimit || !okCur {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return
	}
	page, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), id, limit, cur)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	m, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
