package matching

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	switch {
	case errors.Is(err, ErrSelfSwipe):
		status, code, msg = http.StatusBadRequest, "invalid_target", err.Error()
	case errors.Is(err, ErrProfileIncomplete):
		status, code, msg = http.StatusConflict, "profile_incomplete", err.Error()
	case errors.Is(err, ErrTargetUnavailable):
		status, code, msg = http.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, ErrAlreadySwiped):
		status, code, msg = http.StatusConflict, "already_swiped", err.Error()
	case errors.Is(err, ErrMatchNotFound):
		status, code, msg = http.StatusNotFound, "not_found", err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "targetUserId and action (like|pass) are required")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.TargetUserID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) List(c *gin.Context) {
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), DefaultLimit, MaxLimit), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "matching.list", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Get(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	m, err := h.service.Get(c.Request.Context(), httpx.UserID(c), matchID)
	if err != nil {
		h.fail(c, "matching.get", err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
