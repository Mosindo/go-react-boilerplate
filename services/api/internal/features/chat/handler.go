package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Respond(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), id, httpx.UserID(c)); err != nil {
		httpx.Respond(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	page, err := h.service.Messages(c.Request.Context(), id, httpx.UserID(c), c.Query("before"), httpx.Limit(c, 30, 50))
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
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", ErrEmptyBody.Message)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), id, httpx.UserID(c), req.Body)
	if err != nil {
		httpx.Respond(c, "chat.send", err)
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
		httpx.Respond(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}
