package chat

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrInvalidMessage):
		httpx.BadRequest(c, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func conversationID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "conversation not found")
		return "", false
	}
	return id, true
}

func intQuery(c *gin.Context, name string, max int) (int, bool) {
	raw := c.Query(name)
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		httpx.BadRequest(c, name+" must be between 1 and "+strconv.Itoa(max))
		return 0, false
	}
	return n, true
}

func (h *Handler) Conversations(c *gin.Context) {
	limit, ok := intQuery(c, "limit", MaxListLimit)
	if !ok {
		return
	}
	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.BadRequest(c, "before must be an RFC 3339 timestamp")
			return
		}
		before = &t
	}
	resp, err := h.service.Conversations(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	limit, ok := intQuery(c, "limit", MaxMessageLimit)
	if !ok {
		return
	}
	var before *int64
	if raw := c.Query("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 {
			httpx.BadRequest(c, "before must be a message id")
			return
		}
		before = &n
	}
	resp, err := h.service.Messages(c.Request.Context(), c.GetString("userID"), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "body is required")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	res, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := conversationID(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
