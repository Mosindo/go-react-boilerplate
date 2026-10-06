package profiles

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Self(c *gin.Context) {
	view, err := h.service.Self(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		logger.LogHandlerError(c, "profiles.self", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, toSelfResponse(view))
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SaveProfile(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	h.Self(c)
}

func (h *Handler) SavePrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SavePrivacy(c.Request.Context(), c.GetString("userID"), *req.ShowAge, *req.ShowDistance, *req.Discoverable); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	h.Self(c)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.Self(c)
}

func (h *Handler) SaveLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SaveLocation(c.Request.Context(), c.GetString("userID"), *req.Latitude, *req.Longitude, req.City); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	h.Self(c)
}

func (h *Handler) SaveInterests(c *gin.Context) {
	var req InterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SaveInterests(c.Request.Context(), c.GetString("userID"), req.InterestIDs); err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	h.Self(c)
}

func (h *Handler) Catalog(c *gin.Context) {
	items, err := h.service.Catalog(c.Request.Context())
	if err != nil {
		logger.LogHandlerError(c, "profiles.catalog", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": ToInterestResponses(items, nil)})
}

func (h *Handler) View(c *gin.Context) {
	targetID := c.Param("userId")
	if !uuidPattern.MatchString(targetID) {
		httpx.Error(c, http.StatusNotFound, "profile not found")
		return
	}
	pp, shared, err := h.service.View(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "profile not found")
			return
		}
		logger.LogHandlerError(c, "profiles.view", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.JSON(http.StatusOK, ToPublicResponse(pp, shared))
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnderage):
		httpx.Error(c, http.StatusUnprocessableEntity, "you must be at least 18 years old")
	case errors.Is(err, ErrInvalidFirstName), errors.Is(err, ErrInvalidBirthdate), errors.Is(err, ErrInvalidGender),
		errors.Is(err, ErrBioTooLong), errors.Is(err, ErrInvalidCity), errors.Is(err, ErrInvalidPreferences),
		errors.Is(err, ErrInvalidLocation), errors.Is(err, ErrInvalidInterests):
		httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusConflict, "create your profile first")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func toSelfResponse(v SelfView) SelfResponse {
	resp := SelfResponse{
		Interests:  ToInterestResponses(v.Interests, nil),
		Photos:     ToPhotoResponses(v.Photos),
		Onboarding: OnboardingResponse{Complete: v.Onboarding.Complete, Missing: v.Onboarding.Missing},
	}
	if v.Profile != nil {
		p := v.Profile
		resp.Profile = &ProfileResponse{
			FirstName:    p.FirstName,
			Birthdate:    p.Birthdate.Format("2006-01-02"),
			Age:          AgeOn(p.Birthdate, time.Now()),
			Gender:       p.Gender,
			Bio:          p.Bio,
			City:         p.City,
			HasLocation:  p.HasLocation,
			ShowAge:      p.ShowAge,
			ShowDistance: p.ShowDistance,
			Discoverable: p.Discoverable,
		}
	}
	if v.Preferences != nil {
		resp.Preferences = &PreferencesResponse{
			InterestedIn:  v.Preferences.InterestedIn,
			MinAge:        v.Preferences.MinAge,
			MaxAge:        v.Preferences.MaxAge,
			MaxDistanceKm: v.Preferences.MaxDistanceKm,
		}
	}
	return resp
}
