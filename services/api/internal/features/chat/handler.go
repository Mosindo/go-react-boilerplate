package chat

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, msg = http.StatusNotFound, "conversation not found"
	case errors.Is(err, ErrInvalidBody):
		status, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, ErrInvalidCursor):
		status, msg = http.StatusBadRequest, "invalid cursor"
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, gin.H{"error": msg})
}

func queryInt(c *gin.Context, key string) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + key})
		return 0, false
	}
	return v, true
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := queryInt(c, "limit")
	if !ok {
		return
	}
	out, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, out)
}

func (h *Handler) ListMessages(c *gin.Context) {
	limit, ok := queryInt(c, "limit")
	if !ok {
		return
	}
	before, ok := queryInt(c, "before")
	if !ok {
		return
	}
	out, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("id"), limit, int64(before))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Send(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
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
	receipt, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, receipt)
}

func (h *Handler) Clear(c *gin.Context) {
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
