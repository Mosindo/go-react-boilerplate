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

func (h *Handler) fail(c *gin.Context, err error) {
	if httpx.WriteValidation(c, err) {
		return
	}
	if errors.Is(err, ErrProfileNotFound) {
		httpx.Fail(c, http.StatusConflict, "profile_required", "Crée d'abord ton profil.", err)
		return
	}
	httpx.Internal(c, err)
}

func (h *Handler) GetMine(c *gin.Context) {
	resp, err := h.service.MyProfile(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpsertMine(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpsertProfile(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) ReplaceInterests(c *gin.Context) {
	var req InterestsRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ReplaceInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var req PrivacyRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), httpx.UserID(c), *req.IsVisible, *req.ShowDistance); err != nil {
		h.fail(c, err)
		return
	}
	h.GetMine(c)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.PublicProfile(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), targetID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			httpx.NotFound(c, "Profil introuvable.")
			return
		}
		httpx.Internal(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, h.service.CardResponse(card))
}
