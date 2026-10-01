package discovery

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Error(c, http.StatusConflict, "profile_incomplete", "complétez votre profil (au moins une photo) pour découvrir des profils")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "profil introuvable")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Error(c, http.StatusConflict, "already_swiped", "vous avez déjà répondu à ce profil")
	case errors.Is(err, ErrMatched):
		httpx.Error(c, http.StatusConflict, "matched", "vous avez un match avec cette personne : supprimez le match à la place")
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrSelf):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Feed(c *gin.Context) {
	cards, err := h.service.Feed(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, DefaultBatch, MaxBatch))
	if err != nil {
		fail(c, "discovery.feed", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId et action (like ou pass) sont requis")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		fail(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unswipe(c *gin.Context) {
	if err := h.service.Unswipe(c.Request.Context(), httpx.UserID(c), c.Param("userId")); err != nil {
		fail(c, "discovery.unswipe", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Profile(c *gin.Context) {
	card, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), c.Param("userId"))
	if err != nil {
		fail(c, "discovery.profile", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *Handler) Matches(c *gin.Context) {
	items, err := h.service.Matches(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 30, 100), httpx.Offset(c))
	if err != nil {
		fail(c, "discovery.matches", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"matches": items})
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
