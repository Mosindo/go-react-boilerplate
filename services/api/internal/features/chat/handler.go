package chat

import (
	"errors"
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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrConversationNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "conversation not found", operation, err)
	case errors.Is(err, ErrMessageContentNeeded), errors.Is(err, ErrMessageTooLong), errors.Is(err, ErrSelfConversation):
		httpx.Fail(c, http.StatusBadRequest, "validation_failed", err.Error(), operation, err)
	default:
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not process conversation", operation, err)
	}
}

func otherUserID(c *gin.Context) (string, bool) {
	id := httpx.NormalizeUUID(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(id) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid user id", "chat.validate_user_id", nil)
		return "", false
	}
	return id, true
}

func (h *Handler) Chats(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), DefaultChatsPage, MaxChatsPage)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	chats, err := h.service.ListChats(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ChatsResponse{Chats: chats})
}

func (h *Handler) Messages(c *gin.Context) {
	otherID, ok := otherUserID(c)
	if !ok {
		return
	}
	before := httpx.NormalizeUUID(c.Query("before"))
	if before != "" && !httpx.IsUUID(before) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_cursor", "invalid cursor", "chat.messages.validate", nil)
		return
	}
	limit := httpx.Limit(c.Query("limit"), DefaultMessagesPage, MaxMessagesPage)
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), otherID, limit, before)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	otherID, ok := otherUserID(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "chat.send.bind", err)
		return
	}
	message, err := h.service.Send(c.Request.Context(), httpx.UserID(c), otherID, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

func (h *Handler) MarkRead(c *gin.Context) {
	otherID, ok := otherUserID(c)
	if !ok {
		return
	}
	readAt, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), otherID)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{ReadAt: readAt})
}
