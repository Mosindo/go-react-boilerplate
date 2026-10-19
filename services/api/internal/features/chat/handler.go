package chat

import (
	"errors"
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		platformerrors.Respond(c, http.StatusNotFound, "conversation_not_found", "conversation not found")
	case errors.Is(err, ErrInvalidMessage):
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_message", "message must be 1-2000 characters")
	case errors.Is(err, ErrInvalidCursor):
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid cursor")
	default:
		logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "internal error")
	}
}

func limitParam(c *gin.Context) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid limit")
		return 0, false
	}
	return n, true
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := limitParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetConversation(c *gin.Context) {
	conv, err := h.service.GetConversation(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "chat.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, conv)
}

func (h *Handler) ListMessages(c *gin.Context) {
	limit, ok := limitParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("id"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "chat.send.bind", ErrInvalidMessage)
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

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
