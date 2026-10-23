package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.QueryInt(c, "limit", DefaultLimit, 1, MaxLimit)
	if !ok {
		return
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Profile(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, card)
}
