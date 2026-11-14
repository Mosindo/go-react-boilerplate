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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_required", "complétez votre profil (avec au moins une photo) pour découvrir des profils")
	case errors.Is(err, ErrInvalidSwipe):
		httpx.BadRequest(c, "invalid_swipe", "action invalide")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", "vous avez déjà répondu à ce profil")
	case errors.Is(err, ErrTargetUnavailable):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "ce profil n'est plus disponible")
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "match_not_found", "match introuvable")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	list, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 30))
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if !httpx.IsUUID(target) {
		httpx.BadRequest(c, "invalid_id", "identifiant invalide")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), target, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
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
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
