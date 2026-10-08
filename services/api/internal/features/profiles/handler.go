package profiles

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Me(c *gin.Context) {
	res, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	h.reply(c, res, err)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	res, err := h.service.UpdateProfile(c.Request.Context(), c.GetString("userID"), req)
	h.reply(c, res, err)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	res, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	h.reply(c, res, err)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	res, err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req)
	h.reply(c, res, err)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	res, err := h.service.ClearLocation(c.Request.Context(), c.GetString("userID"))
	h.reply(c, res, err)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) View(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	profile, err := h.service.View(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) reply(c *gin.Context, res MeResponse, err error) {
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}
