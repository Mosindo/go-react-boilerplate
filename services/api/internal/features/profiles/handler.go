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

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.FailInternal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) Me(c *gin.Context) {
	profile, err := h.service.MyProfile(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "profiles.me", profile, err)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, "profiles.update", &req) {
		return
	}
	profile, err := h.service.UpdateProfile(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.update", profile, err)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	profile, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City)
	h.respond(c, "profiles.location", profile, err)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	profile, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.preferences", profile, err)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if !httpx.BindJSON(c, "profiles.settings", &req) {
		return
	}
	profile, err := h.service.UpdateSettings(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.settings", profile, err)
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.ViewProfile(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		if errors.Is(err, ErrNotVisible) || errors.Is(err, ErrProfileNotFound) {
			httpx.Fail(c, http.StatusNotFound, "profile_not_found", "Profil introuvable.")
			return
		}
		httpx.FailInternal(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) respond(c *gin.Context, op string, profile MyProfile, err error) {
	if err != nil {
		if status, code, msg, ok := MapError(err); ok {
			httpx.Fail(c, status, code, msg)
			return
		}
		httpx.FailInternal(c, op, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

// MapError translates domain validation errors into HTTP responses.
func MapError(err error) (int, string, string, bool) {
	switch {
	case errors.Is(err, ErrInvalidFirstName):
		return http.StatusBadRequest, "invalid_first_name", "Prénom invalide (lettres uniquement, 40 caractères max).", true
	case errors.Is(err, ErrUnderage):
		return http.StatusBadRequest, "underage", "Lueur est réservé aux personnes majeures (18 ans et plus).", true
	case errors.Is(err, ErrInvalidBirthdate):
		return http.StatusBadRequest, "invalid_birthdate", "Date de naissance invalide.", true
	case errors.Is(err, ErrInvalidGender):
		return http.StatusBadRequest, "invalid_gender", "Genre invalide.", true
	case errors.Is(err, ErrBioTooLong):
		return http.StatusBadRequest, "bio_too_long", "La bio ne doit pas dépasser 500 caractères.", true
	case errors.Is(err, ErrJobTitleTooLong):
		return http.StatusBadRequest, "job_title_too_long", "Le métier ne doit pas dépasser 80 caractères.", true
	case errors.Is(err, ErrCityTooLong):
		return http.StatusBadRequest, "city_too_long", "La ville ne doit pas dépasser 80 caractères.", true
	case errors.Is(err, ErrTooManyInterests):
		return http.StatusBadRequest, "too_many_interests", "Choisis 10 centres d'intérêt au maximum.", true
	case errors.Is(err, ErrUnknownInterest):
		return http.StatusBadRequest, "unknown_interest", "Centre d'intérêt inconnu.", true
	case errors.Is(err, ErrInvalidLocation):
		return http.StatusBadRequest, "invalid_location", "Position invalide.", true
	case errors.Is(err, ErrInvalidPreferences):
		return http.StatusBadRequest, "invalid_preferences", "Préférences invalides.", true
	case errors.Is(err, ErrProfileIncomplete):
		return http.StatusConflict, "profile_incomplete", "Complète ton profil pour continuer.", true
	case errors.Is(err, ErrProfileNotFound):
		return http.StatusNotFound, "profile_not_found", "Profil introuvable.", true
	}
	return 0, "", "", false
}
