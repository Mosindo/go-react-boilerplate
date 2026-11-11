package profiles

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

func (h *Handler) GetMine(c *gin.Context) {
	resp, err := h.service.GetMine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) respondMine(c *gin.Context) {
	h.GetMine(c)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var req ProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), req); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) SaveInterests(c *gin.Context) {
	var req InterestsRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SaveInterests(c.Request.Context(), httpx.UserID(c), req.Interests); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) SetVisibility(c *gin.Context) {
	var req VisibilityRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetVisibility(c.Request.Context(), httpx.UserID(c), req); err != nil {
		fail(c, err)
		return
	}
	h.respondMine(c)
}

func (h *Handler) Catalog(c *gin.Context) {
	list, err := h.service.Catalog(c.Request.Context())
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, InterestsCatalogResponse{Interests: list})
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.View(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, card)
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidFirstName):
		httpx.BadRequest(c, "invalid_first_name", "Le prénom doit contenir entre 1 et 40 caractères.")
	case errors.Is(err, ErrInvalidBirthdate):
		httpx.BadRequest(c, "invalid_birthdate", "Date de naissance invalide.")
	case errors.Is(err, ErrUnderage):
		httpx.BadRequest(c, "underage", "Tu dois avoir au moins 18 ans pour utiliser l'application.")
	case errors.Is(err, ErrInvalidGender):
		httpx.BadRequest(c, "invalid_gender", "Genre invalide.")
	case errors.Is(err, ErrTextTooLong):
		httpx.BadRequest(c, "text_too_long", "Un des champs est trop long.")
	case errors.Is(err, ErrInvalidPreferences):
		httpx.BadRequest(c, "invalid_preferences", "Préférences invalides (âge 18-99, distance 1-500 km).")
	case errors.Is(err, ErrTooManyInterests):
		httpx.BadRequest(c, "too_many_interests", "Tu peux choisir jusqu'à 10 centres d'intérêt.")
	case errors.Is(err, ErrUnknownInterest):
		httpx.BadRequest(c, "unknown_interest", "Centre d'intérêt inconnu.")
	case errors.Is(err, ErrInvalidLocation):
		httpx.BadRequest(c, "invalid_location", "Position invalide.")
	case errors.Is(err, ErrProfileRequired):
		httpx.Fail(c, http.StatusConflict, "profile_required", "Crée d'abord ton profil.", nil)
	case errors.Is(err, ErrForbidden):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "Profil introuvable.", nil)
	default:
		httpx.Internal(c, err)
	}
}
