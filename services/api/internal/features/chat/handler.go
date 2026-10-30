package chat

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
)

const longPollTimeout = 25 * time.Second

type Handler struct {
	service *Service
	hub     *realtime.Hub
}

func NewHandler(service *Service, hub *realtime.Hub) *Handler {
	return &Handler{service: service, hub: hub}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, op, http.StatusNotFound, "conversation not found", nil)
		return
	}
	httpx.WriteError(c, op, err)
}

func (h *Handler) ListConversations(c *gin.Context) {
	before, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	resp, err := h.service.ListConversations(c.Request.Context(), c.GetString("userID"), before, httpx.QueryLimit(c, DefaultConversation, MaxPageSize))
	if err != nil {
		h.fail(c, "chat.conversations", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMessages(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	before, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMessages(c.Request.Context(), c.GetString("userID"), id, before, httpx.QueryLimit(c, DefaultPageSize, MaxPageSize))
	if err != nil {
		h.fail(c, "chat.messages", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "chat.send.bind", err)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), c.GetString("userID"), id, req.Body)
	if err != nil {
		h.fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Events is a long-poll endpoint. Without ?after= it only returns the current
// cursor; with it, the call holds until something new happens for the caller
// (or ~25s pass) and returns the events after that cursor.
func (h *Handler) Events(c *gin.Context) {
	userID := c.GetString("userID")
	c.Header("Cache-Control", "no-store")

	raw := c.Query("after")
	if raw == "" {
		c.JSON(http.StatusOK, gin.H{"cursor": h.hub.Cursor(), "events": []realtime.Event{}})
		return
	}
	after, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || after < 0 {
		httpx.Fail(c, "chat.events", http.StatusBadRequest, "invalid after cursor", nil)
		return
	}

	events := h.hub.Wait(c.Request.Context(), userID, after, longPollTimeout)
	cursor := after
	if len(events) > 0 {
		cursor = events[len(events)-1].Seq
	}
	if events == nil {
		events = []realtime.Event{}
	}
	c.JSON(http.StatusOK, gin.H{"cursor": cursor, "events": events})
}
