package chat

import (
	"net/http"
	"strconv"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Conversations(c *gin.Context) {
	before := time.Now().Add(time.Minute)
	if raw := c.Query("before"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			httpx.Fail(c, "chat.cursor", platformerrors.Invalid("before must be an RFC3339 timestamp"))
			return
		}
		before = t
	}
	items, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"conversations": items})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var beforeID int64
	if raw := c.Query("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1 {
			httpx.Fail(c, "chat.cursor", platformerrors.Invalid("before must be a message id"))
			return
		}
		beforeID = n
	}
	page, err := h.service.Messages(c.Request.Context(), id, httpx.UserID(c), beforeID, httpx.Limit(c, 50, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), id, httpx.UserID(c), req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), id, httpx.UserID(c)); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), id, httpx.UserID(c)); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
