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

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, "matching.swipe.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	result, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.TargetID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrInvalidTarget):
			httpx.Fail(c, "matching.swipe", http.StatusBadRequest, err.Error(), err)
		case errors.Is(err, ErrNoProfile):
			httpx.Fail(c, "matching.swipe", http.StatusConflict, err.Error(), err)
		case errors.Is(err, ErrTargetUnavailable):
			httpx.Fail(c, "matching.swipe", http.StatusNotFound, "profile not available", err)
		default:
			httpx.Fail(c, "matching.swipe", http.StatusInternalServerError, "internal error", err)
		}
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Matches(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 50)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	matches, err := h.service.Matches(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		httpx.Fail(c, "matching.matches", http.StatusInternalServerError, "internal error", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 30, 50)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	people, total, err := h.service.LikesReceived(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		httpx.Fail(c, "matching.likes", http.StatusInternalServerError, "internal error", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": people, "total": total})
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		httpx.Fail(c, "matching.unmatch.id", http.StatusBadRequest, "invalid match id", errors.New("invalid match id"))
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), matchID); err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			httpx.Fail(c, "matching.unmatch", http.StatusNotFound, "match not found", err)
			return
		}
		httpx.Fail(c, "matching.unmatch", http.StatusInternalServerError, "internal error", err)
		return
	}
	c.Status(http.StatusNoContent)
}
