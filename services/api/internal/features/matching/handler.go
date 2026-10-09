package matching

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplet):
		httpx.Fail(c, http.StatusForbidden, httpx.CodeProfileIncomplete, "complete your profile, photos and preferences first")
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrSelfSwipe):
		httpx.BadRequest(c, "you cannot swipe on yourself")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := 10
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 20 {
			httpx.BadRequest(c, "limit must be between 1 and 20")
			return
		}
		limit = v
	}
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ProfilesResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !req.Action.Valid() {
		httpx.BadRequest(c, "action must be \"like\" or \"pass\"")
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.NotFound(c)
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.UserID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, httpx.DefaultLimit)
	if !ok {
		return
	}
	list, err := h.service.LikesReceived(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "matching.likes", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, ProfilesResponse{Profiles: list})
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, offset, ok := httpx.Pagination(c, httpx.DefaultLimit)
	if !ok {
		return
	}
	list, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "matching.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MatchesResponse{Matches: list})
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
