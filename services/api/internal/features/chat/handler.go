package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 20, 50), c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	page, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, httpx.Limit(c, 30, 50), c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) Read(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Clear(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Clear(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
