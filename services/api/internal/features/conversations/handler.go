package conversations

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
	case errors.Is(err, ErrInvalidMessage):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidMessage.Error()})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func matchParam(c *gin.Context) (string, bool) {
	id := c.Param("matchId")
	if !validate.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return "", false
	}
	return id, true
}

// pageParams parses ?limit= and ?before=<RFC3339>.
func pageParams(c *gin.Context) (limit int, before *time.Time, ok bool) {
	limit = DefaultPageSize
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxPageSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 50"})
			return 0, nil, false
		}
		limit = n
	}
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before must be an RFC3339 timestamp"})
			return 0, nil, false
		}
		before = &t
	}
	return limit, before, true
}

func (h *Handler) List(c *gin.Context) {
	limit, before, ok := pageParams(c)
	if !ok {
		return
	}
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		h.fail(c, "conversations.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: items})
}

func (h *Handler) Messages(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	limit, before, ok := pageParams(c)
	if !ok {
		return
	}
	items, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), matchID, before, limit)
	if err != nil {
		h.fail(c, "conversations.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MessagesResponse{Messages: items})
}

func (h *Handler) Send(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), matchID, req.Body)
	if err != nil {
		h.fail(c, "conversations.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "conversations.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "conversations.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := matchParam(c)
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		h.fail(c, "conversations.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
