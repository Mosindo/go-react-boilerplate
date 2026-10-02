package discovery

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.QueryInt(c, "limit", 10, 1, 20)
	if !ok {
		return
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"profiles": cards})
	case errors.Is(err, ErrIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error())
	default:
		httpx.Internal(c, "discovery.list", err)
	}
}

func (h *Handler) Profile(c *gin.Context) {
	card, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, card)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, "discovery.profile", err)
	}
}
