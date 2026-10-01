package chat

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	if v, ok := httpx.AsValidation(err); ok {
		httpx.Error(c, http.StatusBadRequest, v.Message)
		return
	}
	if errors.Is(err, ErrConversationNotFound) {
		httpx.Error(c, http.StatusNotFound, "conversation not found")
		return
	}
	logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
	httpx.Error(c, http.StatusInternalServerError, "internal error")
}

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 100)
	offset := httpx.Offset(c.Query("offset"))
	list, unread, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, ConversationsResponse{Conversations: list, TotalUnread: unread})
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "conversationId")
	if !ok {
		return
	}
	before := c.Query("before")
	if before != "" && !httpx.IsUUID(before) {
		httpx.Error(c, http.StatusBadRequest, "invalid cursor")
		return
	}
	limit := httpx.Limit(c.Query("limit"), DefaultMessagesPage, MaxMessagesPage)
	msgs, more, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), id, before, limit)
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MessagesResponse{Messages: msgs, HasMore: more})
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "conversationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "chat.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, ReadResponse{Marked: n})
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "chat.clear", err)
		return
	}
	httpx.NoContent(c)
}
