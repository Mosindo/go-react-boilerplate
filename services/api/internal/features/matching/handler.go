package matching

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Discover(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	cards, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toProfilesResponse(cards))
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 20, 50)
	offset := httpx.Offset(c.Query("offset"))
	cards, err := h.service.LikesReceived(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "matching.likes_received", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toProfilesResponse(cards))
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil || !uuidPattern.MatchString(req.UserID) {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	// Canonical lower-case form: ordering, locking and self-checks compare strings.
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), strings.ToLower(req.UserID), req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	resp := SwipeResponse{Action: res.Action, AlreadyProcessed: res.AlreadyProcessed, Matched: res.Matched}
	if res.Match != nil {
		resp.MatchID = res.Match.ID
	}
	status := http.StatusCreated
	if res.AlreadyProcessed {
		status = http.StatusOK
	}
	c.JSON(status, resp)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Coded(c, http.StatusForbidden, "profile_incomplete", err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Error(c, http.StatusNotFound, "profile unavailable")
	case errors.Is(err, ErrSelfSwipe), errors.Is(err, ErrInvalidAction):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func toProfilesResponse(cards []Card) ProfilesResponse {
	out := make([]profiles.PublicProfileResponse, 0, len(cards))
	for _, card := range cards {
		out = append(out, profiles.ToPublicResponse(card.Profile, card.Shared))
	}
	return ProfilesResponse{Profiles: out}
}
