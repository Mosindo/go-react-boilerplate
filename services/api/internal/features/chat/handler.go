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

func writeError(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrInvalidMessage):
		httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrInvalidCursor):
		httpx.Error(c, http.StatusBadRequest, "invalid cursor")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "something went wrong")
	}
}

func conversationID(c *gin.Context) (string, bool) {
	id := c.Param("conversationId")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "conversation not found")
		return "", false
	}
	return id, true
}

func (h *Handler) Conversations(c *gin.Context) {
	limit, ok := httpx.ParseLimit(c.Query("limit"), DefaultPageSize, MaxPageSize)
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "limit must be between 1 and 50")
		return
	}
	resp, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), c.Query("cursor"), limit)
	if err != nil {
		writeError(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	limit, ok := httpx.ParseLimit(c.Query("limit"), DefaultPageSize, MaxPageSize)
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "limit must be between 1 and 50")
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), id, c.Query("cursor"), limit)
	if err != nil {
		writeError(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Content)
	if err != nil {
		writeError(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	readAt, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		writeError(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{ReadAt: readAt})
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		writeError(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
