package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotEligible):
		httpx.Error(c, http.StatusNotFound, "profile is not available")
	case errors.Is(err, ErrMatchNotFound):
		httpx.Error(c, http.StatusNotFound, "match not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), DefaultDiscoverLimit, MaxDiscoverLimit)
	list, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 100)
	offset := httpx.Offset(c.Query("offset"))
	list, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "matching.list_matches", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MatchesResponse{Matches: list})
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.ParamUUID(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	httpx.NoContent(c)
}
