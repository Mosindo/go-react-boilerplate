package chat

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func parseBefore(c *gin.Context) (*time.Time, bool) {
	raw := c.Query("before")
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		httpx.BadRequest(c, "before must be an RFC3339 timestamp")
		return nil, false
	}
	return &t, true
}

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "conversation not found")
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Conversations(c *gin.Context) {
	before, ok := parseBefore(c)
	if !ok {
		return
	}
	resp, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, DefaultPage, MaxPage))
	if err != nil {
		fail(c, "chat.conversations", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	before, ok := parseBefore(c)
	if !ok {
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), c.Param("conversationId"), httpx.UserID(c), before, httpx.Limit(c, DefaultPage, MaxPage))
	if err != nil {
		fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "body is required")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.Param("conversationId"), httpx.UserID(c), req.Body)
	if err != nil {
		fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.Param("conversationId"), httpx.UserID(c)); err != nil {
		fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), c.Param("conversationId"), httpx.UserID(c)); err != nil {
		fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
