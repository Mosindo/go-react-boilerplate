package chat

import (
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) ListConversations(c *gin.Context) {
	page, err := h.service.ListConversations(c.Request.Context(), httpx.UserID(c), c.Query("before"), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Messages(c *gin.Context) {
	var before int64
	if raw := c.Query("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			httpx.BadRequest(c)
			return
		}
		before = n
	}
	page, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), c.Param("id"), before, httpx.Limit(c, 30, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if !httpx.Bind(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}
