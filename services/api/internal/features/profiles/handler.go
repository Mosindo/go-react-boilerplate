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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) validationFail(c *gin.Context, op string, err error) bool {
	codes := map[error]struct {
		status int
		code   string
	}{
		ErrInvalidName:        {http.StatusBadRequest, "invalid_name"},
		ErrInvalidBirthDate:   {http.StatusBadRequest, "invalid_birth_date"},
		ErrUnderage:           {http.StatusForbidden, "underage"},
		ErrBirthDateLocked:    {http.StatusConflict, "birth_date_locked"},
		ErrInvalidGender:      {http.StatusBadRequest, "invalid_gender"},
		ErrInvalidBio:         {http.StatusBadRequest, "invalid_bio"},
		ErrInvalidOccupation:  {http.StatusBadRequest, "invalid_occupation"},
		ErrTooManyInterests:   {http.StatusBadRequest, "too_many_interests"},
		ErrUnknownInterest:    {http.StatusBadRequest, "unknown_interest"},
		ErrInvalidPreferences: {http.StatusBadRequest, "invalid_preferences"},
		ErrInvalidLocation:    {http.StatusBadRequest, "invalid_location"},
		ErrProfileRequired:    {http.StatusConflict, "profile_required"},
	}
	for known, meta := range codes {
		if errors.Is(err, known) {
			httpx.Fail(c, op, meta.status, meta.code, known.Error(), err)
			return true
		}
	}
	return false
}

func (h *Handler) Mine(c *gin.Context) {
	resp, err := h.service.Mine(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Internal(c, "profiles.mine", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, "profiles.upsert", &req) {
		return
	}
	userID := c.GetString("userID")
	if err := h.service.Upsert(c.Request.Context(), userID, req); err != nil {
		if !h.validationFail(c, "profiles.upsert", err) {
			httpx.Internal(c, "profiles.upsert", err)
		}
		return
	}
	h.Mine(c)
}

func (h *Handler) SetPreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	if err := h.service.SetPreferences(c.Request.Context(), c.GetString("userID"), req); err != nil {
		if !h.validationFail(c, "profiles.preferences", err) {
			httpx.Internal(c, "profiles.preferences", err)
		}
		return
	}
	h.Mine(c)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req); err != nil {
		if !h.validationFail(c, "profiles.location", err) {
			httpx.Internal(c, "profiles.location", err)
		}
		return
	}
	h.Mine(c)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var req PrivacyRequest
	if !httpx.BindJSON(c, "profiles.privacy", &req) {
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), c.GetString("userID"), *req.IsDiscoverable, *req.ShowDistance); err != nil {
		if !h.validationFail(c, "profiles.privacy", err) {
			httpx.Internal(c, "profiles.privacy", err)
		}
		return
	}
	h.Mine(c)
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
	targetID, ok := httpx.UUIDParam(c, "profiles.view", "userId")
	if !ok {
		return
	}
	card, err := h.service.View(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, "profiles.view", http.StatusNotFound, "not_found", "profile not found", err)
			return
		}
		httpx.Internal(c, "profiles.view", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, card)
}
