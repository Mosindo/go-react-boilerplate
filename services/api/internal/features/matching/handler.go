package matching

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Discover(c *gin.Context) {
	res, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), defaultPageSize, 20))
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_swipe", "targetUserId (uuid) and action (like|pass) are required")
		return
	}
	req.TargetUserID = strings.ToLower(req.TargetUserID)
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	cards, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 20, 50), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "matching.likes_received", err)
		return
	}
	c.JSON(http.StatusOK, CardsResponse{Profiles: cards})
}

func (h *Handler) Matches(c *gin.Context) {
	list, err := h.service.Matches(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 20, 50), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "matching.matches", err)
		return
	}
	c.JSON(http.StatusOK, MatchesResponse{Matches: list})
}

func (h *Handler) Unmatch(c *gin.Context) {
	otherID := strings.ToLower(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(otherID) {
		httpx.BadRequest(c, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), otherID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPasses(c *gin.Context) {
	n, err := h.service.ResetPasses(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "matching.reset_passes", err)
		return
	}
	c.JSON(http.StatusOK, ResetResponse{Restored: n})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelfSwipe):
		httpx.Fail(c, op, http.StatusBadRequest, "self_swipe", err.Error(), nil)
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, op, http.StatusForbidden, "profile_incomplete", err.Error(), nil)
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Fail(c, op, http.StatusNotFound, "profile_unavailable", err.Error(), nil)
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, op, http.StatusConflict, "already_swiped", err.Error(), nil)
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "match_not_found", err.Error(), nil)
	case errors.Is(err, profiles.ErrProfileNotFound):
		httpx.Fail(c, op, http.StatusUnauthorized, "user_not_found", "user not found", nil)
	default:
		httpx.Internal(c, op, err)
	}
}
