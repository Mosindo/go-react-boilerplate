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

func cursorParam(c *gin.Context) (string, bool) {
	before := strings.ToLower(strings.TrimSpace(c.Query("before")))
	if before != "" && !httpx.IsUUID(before) {
		httpx.Abort(c, http.StatusBadRequest, "invalid_cursor", "invalid cursor")
		return "", false
	}
	return before, true
}

func (h *Handler) ListConversations(c *gin.Context) {
	before, ok := cursorParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), before, httpx.Limit(c.Query("limit"), DefaultConversationsLimit, MaxConversationsLimit))
	if err != nil {
		httpx.Fail(c, "chat.list_conversations", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMessages(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	before, ok := cursorParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), conversationID, before, httpx.Limit(c.Query("limit"), DefaultMessagesLimit, MaxMessagesLimit))
	if err != nil {
		httpx.Fail(c, "chat.list_messages", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.BindJSON(c, "chat.send", &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), conversationID, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), conversationID); err != nil {
		httpx.Fail(c, "chat.mark_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	conversationID, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), conversationID); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
