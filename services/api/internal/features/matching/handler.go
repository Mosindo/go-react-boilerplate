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

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, "matching.swipe", &req) {
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if !httpx.IsUUID(target) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "Identifiant invalide.")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), target, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	status := http.StatusOK
	if res.Matched {
		status = http.StatusCreated
	}
	c.JSON(status, res)
}

func (h *Handler) Undo(c *gin.Context) {
	res, err := h.service.UndoLastPass(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "matching.undo", err)
		return
	}
	c.JSON(http.StatusOK, res)
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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidAction):
		httpx.Fail(c, http.StatusBadRequest, "invalid_action", "Action invalide (like ou pass).")
	case errors.Is(err, ErrCannotSwipeSelf):
		httpx.Fail(c, http.StatusBadRequest, "cannot_swipe_self", "Action impossible sur ton propre profil.")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", "Tu as déjà répondu à ce profil.")
	case errors.Is(err, ErrTargetNotFound):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "Ce profil n'est plus disponible.")
	case errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "match_not_found", "Match introuvable.")
	case errors.Is(err, ErrNothingToUndo):
		httpx.Fail(c, http.StatusNotFound, "nothing_to_undo", "Aucun profil à revoir.")
	default:
		if status, code, msg, ok := profiles.MapError(err); ok {
			httpx.Fail(c, status, code, msg)
			return
		}
		httpx.FailInternal(c, op, err)
	}
}
