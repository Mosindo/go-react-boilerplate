package chat

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrEmptyMessage):
		httpx.BadRequest(c, "message cannot be empty")
	case errors.Is(err, ErrTooLong):
		httpx.BadRequest(c, "message is limited to 2000 characters")
	default:
		logger.LogHandlerError(c, op, 500, err)
		httpx.Internal(c)
	}
}

func conversationID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return "", false
	}
	return id, true
}

func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, 30, 100)
	if !ok {
		return
	}
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"conversations": list})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	limit := 30
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 100 {
			httpx.BadRequest(c, "invalid limit")
			return
		}
		limit = v
	}
	var before *int64
	if raw := c.Query("before"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 1 {
			httpx.BadRequest(c, "invalid before")
			return
		}
		before = &v
	}
	page, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), id, limit, before)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "body is required")
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
	id, ok := conversationID(c)
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
	id, ok := conversationID(c)
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
