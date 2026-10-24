package matching

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit := DefaultDiscoverLimit
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			httpx.Fail(c, "matching.discover", http.StatusBadRequest, "invalid limit", err)
			return
		}
		limit = v
	}
	cards, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		httpx.Internal(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, "matching.swipe", &req) {
		return
	}
	if !httpx.IsUUID(req.TargetID) {
		httpx.Fail(c, "matching.swipe", http.StatusBadRequest, "invalid target", nil)
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), strings.ToLower(req.TargetID), req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotEligible):
			httpx.Fail(c, "matching.swipe", http.StatusNotFound, "profile not available", err)
		case errors.Is(err, ErrAlreadySwiped):
			httpx.Fail(c, "matching.swipe", http.StatusConflict, "you already decided on this profile", err)
		case errors.Is(err, profiles.ErrNotFound):
			httpx.Fail(c, "matching.swipe", http.StatusNotFound, "profile not available", err)
		default:
			httpx.Internal(c, "matching.swipe", err)
		}
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			httpx.Fail(c, "matching.unmatch", http.StatusNotFound, "match not found", err)
			return
		}
		httpx.Internal(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
