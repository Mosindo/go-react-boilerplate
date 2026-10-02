package chat

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageTooLong):
		httpx.Fail(c, http.StatusBadRequest, "invalid_message", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	matchID, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	limit, ok := httpx.QueryInt(c, "limit", 40, 1, 100)
	if !ok {
		return
	}
	var before int64
	if raw := c.Query("before"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 1 {
			httpx.BadRequest(c, "invalid before")
			return
		}
		before = v
	}
	msgs, err := h.service.List(c.Request.Context(), matchID, httpx.UserID(c), before, limit)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "hasMore": len(msgs) == limit})
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_message", "message cannot be empty")
		return
	}
	m, err := h.service.Send(c.Request.Context(), matchID, httpx.UserID(c), req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), matchID, httpx.UserID(c))
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": n})
}

func (h *Handler) Clear(c *gin.Context) {
	matchID, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), matchID, httpx.UserID(c)); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
