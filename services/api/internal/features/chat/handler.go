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

func (h *Handler) Conversations(c *gin.Context) {
	list, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, 50, MaxPage))
	if err != nil {
		httpx.Internal(c, "chat.conversations", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list})
}

func (h *Handler) Messages(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "chat.messages", "invalid before cursor", err)
			return
		}
		before = &t
	}
	resp, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), c.Param("id"), before, httpx.Limit(c, DefaultPage, MaxPage))
	if h.fail(c, "chat.messages", err) {
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "chat.send.bind", "invalid request", err)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), c.Param("id"), req.Body)
	if h.fail(c, "chat.send", err) {
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if h.fail(c, "chat.read", h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if h.fail(c, "chat.hide", h.service.Hide(c.Request.Context(), c.GetString("userID"), c.Param("id"))) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) fail(c *gin.Context, op string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrForbidden):
		httpx.Fail(c, op, http.StatusNotFound, "conversation not found", nil)
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, op, err.Error(), err)
	default:
		httpx.Internal(c, op, err)
	}
	return true
}
