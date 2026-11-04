package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) List(c *gin.Context) {
	resp, err := h.service.Conversations(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "chat.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Messages(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	resp, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), id, c.Query("before"), httpx.Limit(c, 40, 100))
	if err != nil {
		httpx.Fail(c, "chat.messages", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Send(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	var req SendRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	m, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Body)
	if err != nil {
		httpx.Fail(c, "chat.send", err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Hide(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Hide(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.hide", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "chat.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
