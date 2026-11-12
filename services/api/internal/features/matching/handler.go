package matching

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidDecision), errors.Is(err, ErrSelfSwipe):
		httpx.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not process request")
	}
}

// Discover: GET /discover?limit=10&exclude=id1,id2 (exclude = cards still in the client deck).
func (h *Handler) Discover(c *gin.Context) {
	var exclude []string
	for _, id := range strings.Split(c.Query("exclude"), ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if !httpx.IsUUID(id) {
			httpx.Fail(c, http.StatusBadRequest, "invalid exclude id")
			return
		}
		exclude = append(exclude, id)
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), exclude, httpx.Limit(c.Query("limit"), defaultDiscoverLimit, maxDiscoverLimit))
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), httpx.OrganizationID(c), strings.ToLower(req.TargetUserID), strings.TrimSpace(req.Decision))
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMatches(c *gin.Context) {
	matches, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 30, 100), httpx.Offset(c.Query("offset"), 10000))
	if err != nil {
		h.fail(c, "matching.list", err)
		return
	}
	c.JSON(http.StatusOK, MatchesResponse{Matches: matches})
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID := c.Param("matchId")
	if !httpx.IsUUID(matchID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid match id")
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
