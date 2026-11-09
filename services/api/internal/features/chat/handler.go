package chat

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		// Same answer for "does not exist" and "not yours": never reveal other people's conversations.
		httpx.Error(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrEmptyMessage), errors.Is(err, ErrTooLong):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) convID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !uuidRe.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "conversation not found")
		return "", false
	}
	return id, true
}

func (h *Handler) List(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "before must be an RFC3339 timestamp")
			return
		}
		before = &t
	}
	limit := httpx.IntQuery(c, "limit", DefaultPageSize, 1, MaxPageSize)
	list, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"conversations": list})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := h.convID(c)
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !uuidRe.MatchString(before) {
		httpx.Error(c, http.StatusBadRequest, "before must be a message id")
		return
	}
	limit := httpx.IntQuery(c, "limit", DefaultPageSize, 1, MaxPageSize)
	msgs, err := h.service.Messages(c.Request.Context(), id, c.GetString("userID"), before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := h.convID(c)
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), id, c.GetString("userID"), req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := h.convID(c)
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), id, c.GetString("userID")); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := h.convID(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), id, c.GetString("userID")); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
