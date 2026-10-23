package notifications

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
	limit, ok := httpx.QueryInt(c, "limit", DefaultLimit, 1, MaxLimit)
	if !ok {
		return
	}
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c), limit, c.Query("cursor"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
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

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
