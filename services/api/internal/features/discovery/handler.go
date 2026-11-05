package discovery

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrNotEligible):
		httpx.Error(c, http.StatusForbidden, "profile not available")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Feed(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultFeedSize, MaxFeedSize)
	if !ok {
		return
	}
	items, err := h.service.Feed(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		fail(c, "discovery.feed", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": items})
}

func (h *Handler) Profile(c *gin.Context) {
	p, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		fail(c, "discovery.profile", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.Bind(c, &req) {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		fail(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
