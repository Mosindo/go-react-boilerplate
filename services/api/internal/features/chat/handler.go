package chat

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func parseBefore(c *gin.Context) (*time.Time, bool) {
	raw := c.Query("before")
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		httpx.BadRequest(c, "invalid 'before' cursor")
		return nil, false
	}
	return &t, true
}

func (h *Handler) ListConversations(c *gin.Context) {
	before, ok := parseBefore(c)
	if !ok {
		return
	}
	res, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), before, httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "chat.conversations", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMessages(c *gin.Context) {
	before, ok := parseBefore(c)
	if !ok {
		return
	}
	res, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), c.Param("conversationId"), before, httpx.Limit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), c.Param("conversationId"), req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), c.Param("conversationId")); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	if err := h.service.Clear(c.Request.Context(), c.GetString("userID"), c.Param("conversationId")); err != nil {
		httpx.Fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unread(c *gin.Context) {
	n, err := h.service.UnreadTotal(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "chat.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, sends *middleware.Limiter) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/unread-count", requireUser, h.Unread)
	r.GET("/conversations/:conversationId/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:conversationId/messages", requireUser, sends.ByUser(), h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:conversationId", requireUser, h.Clear)
}
