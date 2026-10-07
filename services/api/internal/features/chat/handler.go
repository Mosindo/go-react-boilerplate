package chat

import (
	"net/http"
	"strings"

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
	items, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 20, 50), httpx.Offset(c.Query("offset")))
	if err != nil {
		httpx.Respond(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var before *string
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		if !httpx.IsUUID(raw) {
			httpx.Fail(c, http.StatusBadRequest, "validation_error", "before must be a message id")
			return
		}
		raw = strings.ToLower(raw)
		before = &raw
	}
	page, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, before, httpx.Limit(c.Query("limit"), 30, 100))
	if err != nil {
		httpx.Respond(c, "chat.messages", err)
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
	var req SendMessageRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Content)
	if err != nil {
		httpx.Respond(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) Read(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Respond(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Respond(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
