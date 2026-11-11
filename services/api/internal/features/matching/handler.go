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

func (h *Handler) Discover(c *gin.Context) {
	list, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 25))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.TargetUserID), req.Direction)
	if err != nil {
		fail(c, err)
		return
	}
	status := http.StatusOK
	if resp.Matched {
		status = http.StatusCreated
	}
	c.JSON(status, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", "Complète ton profil (infos, préférences et au moins une photo).", nil)
	case errors.Is(err, ErrSelfSwipe):
		httpx.BadRequest(c, "self_swipe", "Action impossible sur ton propre profil.")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", "Tu as déjà répondu à ce profil.", nil)
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "Ce profil n'est plus disponible.", nil)
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "match_not_found", "Match introuvable.", nil)
	default:
		httpx.Internal(c, err)
	}
}
