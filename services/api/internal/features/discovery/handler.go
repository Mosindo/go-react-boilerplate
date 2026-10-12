package discovery

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Discover(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			httpx.Fail(c, http.StatusBadRequest, "limit must be between 1 and 20")
			return
		}
		limit = n
	}
	cards, err := h.service.Next(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		if errors.Is(err, ErrOnboardingIncomplete) {
			httpx.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		httpx.Internal(c, "discovery.next", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": cards})
}
