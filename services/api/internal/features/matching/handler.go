package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, err error) {
	if httpx.WriteValidation(c, err) {
		return
	}
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", "Complète ton profil (avec au moins une photo) pour découvrir d'autres personnes.", err)
	case errors.Is(err, ErrTargetUnavailable):
		httpx.NotFound(c, "Ce profil n'est plus disponible.")
	case errors.Is(err, ErrMatchNotFound):
		httpx.NotFound(c, "Match introuvable.")
	default:
		httpx.Internal(c, err)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultDiscoveryLimit, MaxDiscoveryLimit)
	if !ok {
		return
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetUserID, req.Action)
	if err != nil {
		h.fail(c, err)
		return
	}
	status := http.StatusCreated
	if resp.AlreadySwiped {
		status = http.StatusOK
	}
	c.JSON(status, resp)
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultMatchesLimit, MaxMatchesLimit)
	if !ok {
		return
	}
	cursor, ok := httpx.DecodeCursor(c)
	if !ok {
		return
	}
	list, next, err := h.service.ListMatches(c.Request.Context(), httpx.UserID(c), cursor, limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, MatchesResponse{Matches: list, NextCursor: next})
}

func (h *Handler) GetMatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	m, err := h.service.GetMatch(c.Request.Context(), httpx.UserID(c), matchID)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, m)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
