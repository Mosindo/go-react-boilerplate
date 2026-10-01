package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// respond maps domain validation errors to 400/409 and anything else to 500.
func respond(c *gin.Context, op string, sp SelfProfile, err error) {
	switch {
	case err == nil:
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, sp)
	case errors.Is(err, ErrUnderage):
		httpx.Error(c, http.StatusUnprocessableEntity, "underage", "vous devez avoir au moins 18 ans")
	case errors.Is(err, ErrProfileMissing):
		httpx.Error(c, http.StatusConflict, "profile_missing", "créez d'abord votre profil")
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidBirthDate), errors.Is(err, ErrInvalidGender),
		errors.Is(err, ErrBioTooLong), errors.Is(err, ErrCityTooLong), errors.Is(err, ErrInvalidPreferences),
		errors.Is(err, ErrInvalidLocation), errors.Is(err, ErrInvalidInterests):
		httpx.BadRequest(c, "données invalides : "+err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Get(c *gin.Context) {
	sp, err := h.service.Get(c.Request.Context(), httpx.UserID(c))
	respond(c, "profiles.get", sp, err)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "prénom, date de naissance et genre sont requis")
		return
	}
	sp, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	respond(c, "profiles.upsert", sp, err)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "préférences invalides")
		return
	}
	sp, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	respond(c, "profiles.preferences", sp, err)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "localisation invalide")
		return
	}
	sp, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req)
	respond(c, "profiles.location", sp, err)
}

func (h *Handler) SetInterests(c *gin.Context) {
	var req InterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "centres d'intérêt invalides")
		return
	}
	sp, err := h.service.SetInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs)
	respond(c, "profiles.interests", sp, err)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.list_interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": items})
}
