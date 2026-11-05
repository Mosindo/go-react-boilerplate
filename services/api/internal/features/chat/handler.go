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

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrInvalidBody):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrInvalidCursor):
		httpx.BadRequest(c, "invalid cursor")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 50)
	if !ok {
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Messages(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 100)
	if !ok {
		return
	}
	var before int64
	if raw := c.Query("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 {
			httpx.BadRequest(c, "invalid before")
			return
		}
		before = n
	}
	msgs, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), c.Param("id"), before, limit)
	if err != nil {
		fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"messages": msgs, "hasMore": len(msgs) == limit})
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if !httpx.Bind(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
