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

func (h *Handler) fail(c *gin.Context, err error) {
	if httpx.WriteValidation(c, err) {
		return
	}
	if errors.Is(err, ErrConversationNotFound) {
		httpx.NotFound(c, "Conversation introuvable.")
		return
	}
	if errors.Is(err, ErrSenderRestricted) {
		httpx.Fail(c, http.StatusForbidden, "account_restricted", "Ton compte est en cours de vérification suite à des signalements.", err)
		return
	}
	httpx.Internal(c, err)
}

func (h *Handler) ListMessages(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	limit, ok := httpx.Limit(c, DefaultMessagesLimit, MaxMessagesLimit)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), matchID, cursor, limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.NoStore(c)
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
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), matchID, req.Content, req.ClientID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	receipt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), matchID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, receipt)
}
