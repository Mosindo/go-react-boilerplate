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
	cards, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), httpx.Limit(c.Query("limit"), defaultDiscoverLimit, maxDiscoverLimit))
	if err != nil {
		if errors.Is(err, ErrProfileIncomplete) {
			httpx.Fail(c, "matching.discover", http.StatusConflict, "profile_incomplete", err.Error(), err)
			return
		}
		httpx.Internal(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, "matching.swipe", &req) {
		return
	}
	if !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, "matching.swipe", http.StatusBadRequest, "invalid_id", "invalid targetUserId", nil)
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.TargetUserID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileIncomplete):
			httpx.Fail(c, "matching.swipe", http.StatusConflict, "profile_incomplete", err.Error(), err)
		case errors.Is(err, ErrSelfSwipe):
			httpx.Fail(c, "matching.swipe", http.StatusBadRequest, "self_swipe", err.Error(), err)
		case errors.Is(err, ErrNotEligible):
			httpx.Fail(c, "matching.swipe", http.StatusNotFound, "not_available", err.Error(), err)
		case errors.Is(err, ErrAlreadySwiped):
			httpx.Fail(c, "matching.swipe", http.StatusConflict, "already_swiped", err.Error(), err)
		default:
			httpx.Internal(c, "matching.swipe", err)
		}
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Matches(c *gin.Context) {
	matches, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"),
		httpx.Limit(c.Query("limit"), defaultMatchesLimit, maxMatchesLimit), httpx.Offset(c.Query("offset")))
	if err != nil {
		httpx.Internal(c, "matching.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MatchesResponse{Matches: matches})
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matching.unmatch", "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			httpx.Fail(c, "matching.unmatch", http.StatusNotFound, "not_found", "match not found", err)
			return
		}
		httpx.Internal(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
