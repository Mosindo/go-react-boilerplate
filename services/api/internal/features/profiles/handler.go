package profiles

import (
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
		httpx.Fail(c, "profiles.get_mine", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, "profiles.upsert", &req) {
		return
	}
	resp, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	resp, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.BindJSON(c, "profiles.privacy", &req) {
		return
	}
	resp, err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.privacy", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	resp, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListInterests(c *gin.Context) {
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, InterestsResponse{Interests: list})
}

func (h *Handler) ReplaceInterests(c *gin.Context) {
	var req UpdateInterestsRequest
	if !httpx.BindJSON(c, "profiles.replace_interests", &req) {
		return
	}
	list, err := h.service.ReplaceInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs)
	if err != nil {
		httpx.Fail(c, "profiles.replace_interests", err)
		return
	}
	c.JSON(http.StatusOK, InterestsResponse{Interests: list})
}

func (h *Handler) GetCard(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.GetCard(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		httpx.Fail(c, "profiles.card", err)
		return
	}
	c.JSON(http.StatusOK, card)
}
