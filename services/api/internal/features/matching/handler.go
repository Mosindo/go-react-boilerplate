package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeError(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.ErrorCode(c, http.StatusConflict, "profile_incomplete", err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrMatchNotFound):
		httpx.Error(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.ErrorCode(c, http.StatusConflict, "already_swiped", err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "something went wrong")
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.ParseLimit(c.Query("limit"), DefaultDiscoverLimit, MaxDiscoverLimit)
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "limit must be between 1 and 20")
		return
	}
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		writeError(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.UserID, req.Action)
	if err != nil {
		writeError(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Profile(c *gin.Context) {
	target := c.Param("userId")
	if !httpx.IsUUID(target) {
		httpx.Error(c, http.StatusNotFound, "not found")
		return
	}
	p, err := h.service.Profile(c.Request.Context(), c.GetString("userID"), target)
	if err != nil {
		writeError(c, "matching.profile", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ResetPasses(c *gin.Context) {
	n, err := h.service.ResetPasses(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "matching.reset_passes", err)
		return
	}
	c.JSON(http.StatusOK, ResetPassesResponse{Reset: n})
}

func (h *Handler) Unmatch(c *gin.Context) {
	id := c.Param("matchId")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "not found")
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		writeError(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
