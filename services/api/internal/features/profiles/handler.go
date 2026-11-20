package profiles

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "profile not created yet")
	case errors.Is(err, ErrProfileNotVisible):
		httpx.Fail(c, http.StatusNotFound, "not_found", "profile not available")
	case errors.Is(err, ErrUnderage):
		httpx.Fail(c, http.StatusUnprocessableEntity, "underage", err.Error())
	case errors.Is(err, ErrBirthdateLocked):
		httpx.Fail(c, http.StatusUnprocessableEntity, "birthdate_locked", err.Error())
	case errors.Is(err, ErrInvalidFirstName), errors.Is(err, ErrInvalidBirthdate), errors.Is(err, ErrInvalidGender),
		errors.Is(err, ErrInvalidText), errors.Is(err, ErrInvalidHeight), errors.Is(err, ErrInvalidGoal),
		errors.Is(err, ErrInvalidInterests), errors.Is(err, ErrInvalidPrefs), errors.Is(err, ErrInvalidLocation):
		httpx.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
	default:
		httpx.FailInternal(c, operation, err)
	}
}

func toProfileResponse(p Profile) ProfileResponse {
	return ProfileResponse{
		UserID:           p.UserID,
		FirstName:        p.FirstName,
		Birthdate:        p.Birthdate.Format(birthdateLayout),
		Age:              AgeOn(p.Birthdate, time.Now()),
		Gender:           p.Gender,
		Bio:              p.Bio,
		JobTitle:         p.JobTitle,
		City:             p.City,
		HeightCm:         p.HeightCm,
		RelationshipGoal: p.RelationshipGoal,
		HasLocation:      p.Latitude != nil,
		Discoverable:     p.Discoverable,
		Interests:        p.Interests,
		Photos:           p.Photos,
		Preferences:      p.Preferences,
	}
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

func (h *Handler) GetOwn(c *gin.Context) {
	profile, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(profile))
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, "profiles.upsert", &req) {
		return
	}
	profile, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), UpsertProfileInput(req))
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, toProfileResponse(profile))
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), Preferences(req))
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, "profiles.location_clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateVisibility(c *gin.Context) {
	var req VisibilityRequest
	if !httpx.BindJSON(c, "profiles.visibility", &req) {
		return
	}
	if err := h.service.SetDiscoverable(c.Request.Context(), httpx.UserID(c), *req.Discoverable); err != nil {
		h.fail(c, "profiles.visibility", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.PathUUID(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.ViewProfile(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, ToCardResponse(card))
}
