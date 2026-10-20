package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) ListConversations(c *gin.Context) {
	limit, ok := httpx.Limit(c, 20, 50)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeTimeCursor(c.Query("cursor"))
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	res, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), cursor, limit)
	if err != nil {
		httpx.Respond(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMessages(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 50)
	if !ok {
		return
	}
	before, ok := ParseBefore(c.Query("cursor"))
	if !ok {
		httpx.Fail(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	res, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), c.Param("id"), before, limit)
	if err != nil {
		httpx.Respond(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		httpx.Respond(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Respond(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Respond(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
