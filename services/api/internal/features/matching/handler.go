package matching

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "matching.swipe.bind", err)
		return
	}
	targetID := httpx.NormalizeUUID(req.TargetUserID)
	if !httpx.IsUUID(targetID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid user id", "matching.swipe.validate", nil)
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), targetID, req.Decision)
	if err != nil {
		switch {
		case errors.Is(err, ErrSelfSwipe):
			httpx.Fail(c, http.StatusBadRequest, "invalid_target", err.Error(), "matching.swipe", err)
		case errors.Is(err, ErrAlreadySwiped):
			httpx.Fail(c, http.StatusConflict, "already_swiped", err.Error(), "matching.swipe", err)
		case errors.Is(err, ErrNotEligible):
			httpx.Fail(c, http.StatusNotFound, "not_found", err.Error(), "matching.swipe", err)
		case errors.Is(err, ErrProfileRequired):
			httpx.Fail(c, http.StatusConflict, "profile_required", err.Error(), "matching.swipe", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not save decision", "matching.swipe", err)
		}
		return
	}
	status := http.StatusOK
	if resp.Matched {
		status = http.StatusCreated
	}
	c.JSON(status, resp)
}

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 100)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	matches, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not load matches", "matching.list", err)
		return
	}
	c.JSON(http.StatusOK, MatchesResponse{Matches: matches})
}

func (h *Handler) Unmatch(c *gin.Context) {
	otherID := httpx.NormalizeUUID(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(otherID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid user id", "matching.unmatch.validate", nil)
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), otherID); err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "match not found", "matching.unmatch", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not unmatch", "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
