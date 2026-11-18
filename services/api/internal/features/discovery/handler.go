package discovery

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Feed(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), DefaultFeedSize, MaxFeedSize)
	cards, err := h.service.Feed(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		if errors.Is(err, ErrProfileRequired) {
			httpx.Fail(c, http.StatusConflict, "profile_required", err.Error(), "discovery.feed", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not load profiles", "discovery.feed", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, FeedResponse{Profiles: cards})
}

func (h *Handler) Profile(c *gin.Context) {
	targetID := httpx.NormalizeUUID(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(targetID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid user id", "discovery.profile.validate", nil)
		return
	}
	card, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "profile not found", "discovery.profile", err)
			return
		}
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not load profile", "discovery.profile", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, card)
}
