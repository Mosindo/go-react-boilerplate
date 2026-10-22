package chat

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrConversationNotFound):
		status, body = http.StatusNotFound, gin.H{"error": "conversation not found"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func parseCursor(c *gin.Context) (*time.Time, bool) {
	raw := c.Query("cursor")
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
		return nil, false
	}
	return &t, true
}

func (h *Handler) ListConversations(c *gin.Context) {
	before, ok := parseCursor(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	resp, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		writeError(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMessages(c *gin.Context) {
	before, ok := parseCursor(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	resp, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("id"), before, limit)
	if err != nil {
		writeError(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), c.Param("id"), req.Content)
	if err != nil {
		writeError(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	count, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		writeError(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": count})
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		writeError(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
