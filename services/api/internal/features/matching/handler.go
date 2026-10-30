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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, op, http.StatusConflict, err.Error(), nil)
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "not found", nil)
	default:
		httpx.WriteError(c, op, err)
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := httpx.QueryLimit(c, DefaultDiscoverLimit, MaxDiscoverLimit)
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "matching.swipe.bind", err)
		return
	}
	if !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, "matching.swipe", http.StatusBadRequest, "invalid targetUserId", nil)
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
