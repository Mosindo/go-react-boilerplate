package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case errors.Is(err, ErrNoProfile):
		httpx.Error(c, http.StatusConflict, "profile_missing", err.Error())
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "profile not found")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Mine(c *gin.Context) {
	resp, err := h.service.Mine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.mine", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var req ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "firstName, birthDate, gender, latitude and longitude are required")
		return
	}
	in, err := h.service.ValidateProfile(ProfileInput{
		FirstName: req.FirstName, Gender: req.Gender, Bio: req.Bio, City: req.City,
		Latitude: *req.Latitude, Longitude: *req.Longitude, Interests: req.Interests,
	}, req.BirthDate)
	if err != nil {
		h.fail(c, "profiles.validate", err)
		return
	}
	if err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in); err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	h.Mine(c)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "interestedIn, minAge, maxAge and maxDistanceKm are required")
		return
	}
	prefs, err := h.service.ValidatePreferences(Preferences{
		InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm,
	})
	if err != nil {
		h.fail(c, "profiles.prefs.validate", err)
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), prefs); err != nil {
		h.fail(c, "profiles.prefs.save", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "discoverable and showDistance are required")
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), httpx.UserID(c), *req.Discoverable, *req.ShowDistance); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	h.Mine(c)
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) View(c *gin.Context) {
	p, err := h.service.View(c.Request.Context(), httpx.UserID(c), c.Param("userId"))
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}
