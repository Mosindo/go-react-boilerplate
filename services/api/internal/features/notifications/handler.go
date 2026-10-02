package notifications

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.QueryInt(c, "limit", 30, 1, 50)
	if !ok {
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), limit)
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, page)
	case errors.Is(err, ErrBadCursor):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, "notifications.list", err)
	}
}

func (h *Handler) Summary(c *gin.Context) {
	s, err := h.service.Summary(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "notifications.summary", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, s)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	switch err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id); {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, "notifications.read", err)
	}
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "notifications.read_all", err)
		return
	}
	c.Status(http.StatusNoContent)
}
