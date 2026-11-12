package chat

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotMatched):
		// 404 rather than 403: never confirm that a conversation exists.
		httpx.Fail(c, http.StatusNotFound, "conversation not found")
	case errors.Is(err, ErrMessageContentNeeded), errors.Is(err, ErrMessageTooLong), errors.Is(err, ErrInvalidCursor):
		httpx.Fail(c, http.StatusBadRequest, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not process chat request")
	}
}

func otherUser(c *gin.Context) (string, bool) {
	other := strings.ToLower(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(other) {
		httpx.Fail(c, http.StatusBadRequest, "invalid user id")
		return "", false
	}
	if other == httpx.UserID(c) {
		httpx.Fail(c, http.StatusBadRequest, "cannot chat with yourself")
		return "", false
	}
	return other, true
}

func (h *Handler) Chats(c *gin.Context) {
	chats, err := h.service.ListChats(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10000))
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ChatsResponse{Chats: chats})
}

func (h *Handler) Unread(c *gin.Context) {
	n, err := h.service.UnreadConversations(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "chat.unread", err)
		return
	}
	c.JSON(http.StatusOK, UnreadResponse{UnreadConversations: n})
}

func (h *Handler) Messages(c *gin.Context) {
	other, ok := otherUser(c)
	if !ok {
		return
	}
	before := strings.TrimSpace(c.Query("before"))
	if before != "" && !httpx.IsUUID(before) {
		h.fail(c, "chat.messages", ErrInvalidCursor)
		return
	}
	res, err := h.service.ListMessages(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), other, before, httpx.Limit(c.Query("limit"), defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Send(c *gin.Context) {
	other, ok := otherUser(c)
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	msg, err := h.service.SendMessage(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), other, req.Content)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	other, ok := otherUser(c)
	if !ok {
		return
	}
	res, err := h.service.MarkRead(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), other)
	if err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Hide(c *gin.Context) {
	other, ok := otherUser(c)
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), other); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
