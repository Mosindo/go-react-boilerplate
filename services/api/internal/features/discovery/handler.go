package discovery

import (
	"errors"
	"net/http"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service  *Service
	profiles *profiles.Service
}

func NewHandler(service *Service, profilesSvc *profiles.Service) *Handler {
	return &Handler{service: service, profiles: profilesSvc}
}

func (h *Handler) Deck(c *gin.Context) {
	exclude, err := ParseExclude(c.Query("exclude"))
	if err != nil {
		httpx.BadRequest(c, "invalid_exclude", err.Error())
		return
	}
	cards, err := h.service.Deck(c.Request.Context(), httpx.UserID(c), exclude, httpx.Limit(c, DefaultDeckSize, MaxDeckSize))
	if err != nil {
		h.fail(c, "discovery.deck", err)
		return
	}
	out := make([]profiles.CardResponse, 0, len(cards))
	for _, card := range cards {
		out = append(out, h.profiles.ToCardResponse(card))
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, DeckResponse{Profiles: out})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "targetId (uuid) and action (like|pass) are required")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetID, req.Action)
	if err != nil {
		h.fail(c, "discovery.swipe", err)
		return
	}
	resp := SwipeResponse{Matched: res.Matched}
	if res.Matched {
		resp.MatchID = &res.MatchID
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error())
	case errors.Is(err, ErrSuspended):
		httpx.Fail(c, http.StatusForbidden, "suspended", err.Error())
	case errors.Is(err, ErrNotAvailable):
		httpx.Fail(c, http.StatusNotFound, "profile_unavailable", err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", err.Error())
	case errors.Is(err, ErrSelfSwipe):
		httpx.BadRequest(c, "invalid_target", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
