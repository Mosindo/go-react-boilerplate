package chat

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	switch {
	case errors.Is(err, ErrConversationNotFound):
		// 404 rather than 403: do not reveal whether a conversation exists.
		status, code, msg = http.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, ErrCursorNotFound):
		status, code, msg = http.StatusBadRequest, "invalid_cursor", err.Error()
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrMessageTooLong):
		status, code, msg = http.StatusBadRequest, "invalid_message", err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

func (h *Handler) List(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_cursor", "invalid cursor")
		return
	}
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), matchID, before, httpx.Limit(c.Query("limit"), DefaultLimit, MaxLimit))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_message", "message body is required")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), matchID, req.Body, req.ClientMessageID)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
