package profiles

import (
	"net/http"
	"regexp"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID string; handlers use it to reject
// malformed path parameters before they reach the database.
func IsUUID(s string) bool { return uuidPattern.MatchString(s) }

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) toResponse(p Profile) ProfileResponse {
	return ProfileResponse{
		UserID:       p.UserID,
		FirstName:    p.FirstName,
		BirthDate:    p.BirthDate.Format("2006-01-02"),
		Age:          AgeAt(p.BirthDate, h.service.Now()),
		Gender:       p.Gender,
		Bio:          p.Bio,
		City:         p.City,
		HasLocation:  p.HasLocation,
		IsVisible:    p.IsVisible,
		ShowDistance: p.ShowDistance,
		Interests:    p.Interests,
		Photos:       p.Photos,
	}
}

func (h *Handler) GetProfile(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "profiles.get", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, h.toResponse(p))
}

func (h *Handler) PutProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "profiles.put", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, h.toResponse(p))
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "profiles.preferences.get", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, PreferencesResponse(p))
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.SetPreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "profiles.preferences.put", err)
		return
	}
	c.JSON(http.StatusOK, PreferencesResponse(p))
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) View(c *gin.Context) {
	id := c.Param("id")
	if !IsUUID(id) {
		httpx.Fail(c, "profiles.view", platformerrors.NotFound("profile not found"))
		return
	}
	p, err := h.service.View(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		httpx.Fail(c, "profiles.view", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}
