package matching

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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.FailCode(c, operation, http.StatusConflict, "profile_incomplete", err.Error(), err)
	case errors.Is(err, ErrAlreadySwiped):
		httpx.FailCode(c, operation, http.StatusConflict, "already_swiped", err.Error(), err)
	case errors.Is(err, ErrSelfSwipe):
		httpx.FailCode(c, operation, http.StatusBadRequest, "validation", err.Error(), err)
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, operation, http.StatusNotFound, err.Error(), err)
	default:
		httpx.Fail(c, operation, http.StatusInternalServerError, "could not process request", err)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.OrganizationID(c),
		httpx.Limit(c.Query("limit"), defaultDiscoveryLimit, maxDiscoveryLimit))
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "matching.swipe.bind", "targetUserId and action (like|pass) are required", err)
		return
	}
	result, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetUserID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Unmatch(c *gin.Context) {
	other, ok := httpx.UUIDParam(c, "userId", "matching.unmatch.validate")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), other); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
