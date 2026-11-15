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

func (h *Handler) List(c *gin.Context) {
	cursor, err := httpx.DecodeCursor(c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	resp, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), cursor, httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	resp, err := h.service.GetConversation(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.get", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	cursor, err := httpx.DecodeCursor(c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), httpx.UserID(c), id, cursor, httpx.Limit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	var req SendMessageRequest
	if !httpx.Bind(c, "chat.send", &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	at, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"readAt": at})
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "conversationId")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}
