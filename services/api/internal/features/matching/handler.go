package matching

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
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
	case errors.Is(err, ErrProfileIncomplete), errors.Is(err, profiles.ErrProfileIncomplete):
		platformerrors.Respond(c, http.StatusConflict, "profile_incomplete", err.Error())
	case errors.Is(err, ErrTargetUnavailable):
		platformerrors.Respond(c, http.StatusNotFound, "profile_unavailable", "profile unavailable")
	case errors.Is(err, ErrMatchNotFound):
		platformerrors.Respond(c, http.StatusNotFound, "match_not_found", "match not found")
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrInvalidCursor):
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid request")
	default:
		logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "internal error")
	}
}

func limitParam(c *gin.Context) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid limit")
		return 0, false
	}
	return n, true
}

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := limitParam(c)
	if !ok {
		return
	}
	resp, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "matching.swipe.bind", ErrInvalidAction)
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.UserID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListMatches(c *gin.Context) {
	limit, ok := limitParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "matching.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReceivedLikes(c *gin.Context) {
	limit, ok := limitParam(c)
	if !ok {
		return
	}
	resp, err := h.service.ReceivedLikes(c.Request.Context(), c.GetString("userID"), limit, c.Query("cursor"))
	if err != nil {
		h.fail(c, "matching.likes", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}
