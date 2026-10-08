package discovery

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, err := pagination.Limit(c.Query("limit"), DefaultBatch, MaxBatch)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Matches(c *gin.Context) {
	limit, err := pagination.Limit(c.Query("limit"), 30, 50)
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	res, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
