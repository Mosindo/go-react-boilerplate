package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	items, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, DefaultBatch, MaxBatch))
	if err != nil {
		httpx.Internal(c, "matching.discover", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: items})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "targetId and action (like or pass) are required")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetID, req.Action)
	switch {
	case errors.Is(err, ErrSelfSwipe), errors.Is(err, ErrInvalidAction):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrNoProfile):
		httpx.Error(c, http.StatusConflict, "profile_missing", err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Error(c, http.StatusNotFound, "not_found", "profile unavailable")
	case err != nil:
		httpx.Internal(c, "matching.swipe", err)
	default:
		c.JSON(http.StatusOK, resp)
	}
}

func (h *Handler) ResetPasses(c *gin.Context) {
	n, err := h.service.ResetPasses(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "matching.reset_passes", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"restored": n})
}

func (h *Handler) Unmatch(c *gin.Context) {
	err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("matchId"))
	switch {
	case errors.Is(err, ErrMatchNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "match not found")
	case err != nil:
		httpx.Internal(c, "matching.unmatch", err)
	default:
		c.Status(http.StatusNoContent)
	}
}
