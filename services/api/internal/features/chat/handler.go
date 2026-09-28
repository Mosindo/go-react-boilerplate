package chat

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Messages(c *gin.Context) {
	limit, err := httpx.ParseLimit(c, DefaultLimit, MaxLimit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page, err := h.service.Messages(c.Request.Context(), httpx.UserID(c), c.Param("id"), c.Query("before"), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	msg, err := h.service.Send(c.Request.Context(), httpx.UserID(c), c.Param("id"), req.Body)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) MarkRead(c *gin.Context) {
	if err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
