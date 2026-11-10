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

func (h *Handler) GetOwn(c *gin.Context) {
	own, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "profiles.get_own", err)
		return
	}
	resp := MeProfileResponse{
		Photos:     h.service.ToPhotoResponses(own.Photos),
		Interests:  own.Interests,
		Completion: CompletionResponse{Complete: len(own.Missing) == 0, Missing: own.Missing},
	}
	if resp.Interests == nil {
		resp.Interests = []Interest{}
	}
	if p := own.Profile; p != nil {
		resp.Profile = &OwnProfileResponse{
			FirstName: p.FirstName, Birthdate: p.Birthdate.Format("2006-01-02"),
			Age: AgeAt(p.Birthdate, h.service.now()), Gender: p.Gender, Bio: p.Bio, JobTitle: p.JobTitle,
			City: p.City, HeightCm: p.HeightCm, RelationshipGoal: p.RelationshipGoal,
			HasLocation: p.Latitude != nil, IsDiscoverable: p.IsDiscoverable, ShowDistance: p.ShowDistance,
			ModerationStatus: p.ModerationStatus,
		}
	}
	if prefs := own.Preferences; prefs != nil {
		resp.Preferences = &PreferencesResponse{
			InterestedIn: prefs.InterestedIn, MinAge: prefs.MinAge, MaxAge: prefs.MaxAge, MaxDistanceKm: prefs.MaxDistanceKm,
		}
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "first name, birthdate (YYYY-MM-DD) and gender are required")
		return
	}
	if err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	h.GetOwn(c)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "latitude and longitude are required")
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_preferences", "interestedIn, minAge, maxAge and maxDistanceKm are required")
		return
	}
	if err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "isDiscoverable and showDistance are required")
		return
	}
	if err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), *req.IsDiscoverable, *req.ShowDistance); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.View(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, h.service.ToCardResponse(card))
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Fail(c, http.StatusConflict, "profile_required", "create your profile first")
	case errors.Is(err, ErrNotVisible):
		httpx.Fail(c, http.StatusNotFound, "profile_unavailable", "this profile is not available")
	case errors.Is(err, ErrUnderage):
		httpx.BadRequest(c, "underage", err.Error())
	case errors.Is(err, ErrInvalidFirstName), errors.Is(err, ErrInvalidBirthdate), errors.Is(err, ErrBirthdateLocked),
		errors.Is(err, ErrInvalidGender), errors.Is(err, ErrInvalidGoal), errors.Is(err, ErrInvalidHeight),
		errors.Is(err, ErrTooLong), errors.Is(err, ErrTooManyInterests), errors.Is(err, ErrUnknownInterest),
		errors.Is(err, ErrInvalidLocation), errors.Is(err, ErrInvalidPreferences):
		httpx.BadRequest(c, "validation_failed", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
