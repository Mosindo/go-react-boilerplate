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
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	limit, ok := httpx.QueryInt(c, "limit", DefaultLimit, 1, MaxLimit)
	if !ok {
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), id, limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
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
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), id, req.Content)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
