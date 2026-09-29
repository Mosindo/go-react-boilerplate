package chat

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool { return uuidRe.MatchString(s) }

// parseLimit returns 0 (meaning default) when absent and an error for malformed/negative values.
func parseLimit(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
	case errors.Is(err, ErrBlocked):
		c.JSON(http.StatusForbidden, gin.H{"error": ErrBlocked.Error()})
	case errors.Is(err, ErrInvalidBody):
		badRequest(c, ErrInvalidBody.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := parseLimit(c.Query("limit"))
	if !ok {
		badRequest(c, "invalid limit")
		return
	}
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			badRequest(c, "invalid before cursor")
			return
		}
		before = &t
	}
	var beforeID *string
	if raw := c.Query("beforeId"); raw != "" {
		if !isUUID(raw) || before == nil {
			badRequest(c, "invalid beforeId cursor")
			return
		}
		beforeID = &raw
	}
	list, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), before, beforeID, limit)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.JSON(http.StatusOK, ListConversationsResponse{Conversations: list})
}

func (h *Handler) ListMessages(c *gin.Context) {
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	limit, ok := parseLimit(c.Query("limit"))
	if !ok {
		badRequest(c, "invalid limit")
		return
	}
	var before *string
	if raw := c.Query("before"); raw != "" {
		if !isUUID(raw) {
			badRequest(c, "invalid before cursor")
			return
		}
		before = &raw
	}
	resp, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SendMessage(c *gin.Context) {
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid request body")
		return
	}
	msg, err := h.service.SendMessage(c.Request.Context(), c.GetString("userID"), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
