package profiles

import (
	"net/http"

	perr "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.svc.Interests(c.Request.Context())
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Get(c *gin.Context) {
	b, err := h.svc.Get(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, b)
}

func (h *Handler) Put(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Requête invalide."))
		return
	}
	b, err := h.svc.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Requête invalide."))
		return
	}
	b, err := h.svc.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, b)
}
