package chat

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Conversations(c *gin.Context) {
	resp, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 20, 50), c.Query("cursor"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), matchID, httpx.Limit(c, 30, 100), c.Query("cursor"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), matchID, req.Body)
	if err != nil {
		fail(c, err)
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
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotMember):
		// 404 rather than 403: do not reveal whether the conversation exists.
		httpx.Fail(c, http.StatusNotFound, "conversation_not_found", "Conversation introuvable.", nil)
	case errors.Is(err, ErrEmptyMessage):
		httpx.BadRequest(c, "empty_message", "Le message est vide.")
	case errors.Is(err, ErrMessageTooLong):
		httpx.BadRequest(c, "message_too_long", "Message trop long (2000 caractères maximum).")
	case errors.Is(err, ErrInvalidCursor):
		httpx.BadRequest(c, "invalid_cursor", "Pagination invalide.")
	default:
		httpx.Internal(c, err)
	}
}
