package chat

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 100)
	if !ok {
		return
	}
	var before *time.Time
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Fail(c, "chat.list", platformerrors.Invalid("before must be an RFC3339 timestamp"))
			return
		}
		before = &t
	}
	resp, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMessages(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultPageSize, MaxPageSize)
	if !ok {
		return
	}
	var before *int64
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 {
			httpx.Fail(c, "chat.messages", platformerrors.Invalid("before must be a message id"))
			return
		}
		before = &n
	}
	resp, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("id"), before, limit)
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), c.Param("id"), req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	resp, err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
