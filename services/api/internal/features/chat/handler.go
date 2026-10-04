package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func nextCursor(next string) *string {
	if next == "" {
		return nil
	}
	return &next
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, err := httpx.Limit(c)
	if err != nil {
		httpx.Fail(c, "matches.list", err)
		return
	}
	items, next, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "matches.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"matches": items, "nextCursor": nextCursor(next)})
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "matches.unmatch", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ListConversations(c *gin.Context) {
	limit, err := httpx.Limit(c)
	if err != nil {
		httpx.Fail(c, "conversations.list", err)
		return
	}
	items, next, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "conversations.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": items, "nextCursor": nextCursor(next)})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	n, err := h.service.UnreadTotal(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "conversations.unread", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unreadCount": n})
}

func (h *Handler) Messages(c *gin.Context) {
	limit, err := httpx.LimitWith(c, 30, 50)
	if err != nil {
		httpx.Fail(c, "messages.list", err)
		return
	}
	items, next, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), c.Param("id"), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "messages.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": items, "nextCursor": nextCursor(next)})
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		httpx.Fail(c, "messages.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "conversations.read", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "conversations.hide", err)
		return
	}
	httpx.NoContent(c)
}
