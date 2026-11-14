package chat

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		// 404 (not 403) so non-members cannot probe which conversations exist.
		httpx.Fail(c, http.StatusNotFound, "conversation_not_found", "conversation introuvable")
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, "invalid_message", "le message doit contenir entre 1 et 2000 caractères")
	case errors.Is(err, ErrInvalidCursor):
		httpx.BadRequest(c, "invalid_cursor", "curseur de pagination invalide")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Conversations(c *gin.Context) {
	list, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 100), httpx.Offset(c, 10_000))
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list})
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.BadRequest(c, "invalid_cursor", "curseur de pagination invalide")
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), matchID, before, httpx.Limit(c, 30, 100))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), matchID, req.Body)
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
	readAt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), matchID)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{ReadAt: readAt})
}

func (h *Handler) Clear(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
