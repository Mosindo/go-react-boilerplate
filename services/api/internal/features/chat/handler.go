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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "conversation not found", err)
	case errors.Is(err, ErrInvalidMessage):
		httpx.Fail(c, op, http.StatusBadRequest, err.Error(), err)
	default:
		httpx.Fail(c, op, http.StatusInternalServerError, "internal error", err)
	}
}

func matchParam(c *gin.Context, op string) (string, bool) {
	id, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		httpx.Fail(c, op, http.StatusBadRequest, "invalid conversation id", errors.New("invalid conversation id"))
	}
	return id, ok
}

func (h *Handler) Conversations(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 50)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	list, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"conversations": list})
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := matchParam(c, "chat.messages.id")
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, "chat.messages.before", http.StatusBadRequest, "invalid cursor", errors.New("invalid cursor"))
		return
	}
	page, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), matchID, before, httpx.Limit(c.Query("limit"), 30, 100))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := matchParam(c, "chat.send.id")
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, "chat.send.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	m, err := h.service.Send(c.Request.Context(), c.GetString("userID"), matchID, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := matchParam(c, "chat.read.id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	matchID, ok := matchParam(c, "chat.clear.id")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unread(c *gin.Context) {
	n, err := h.service.UnreadTotal(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "chat.unread", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"unread": n})
}
