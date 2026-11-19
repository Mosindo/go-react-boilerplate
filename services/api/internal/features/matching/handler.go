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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, ok := httpx.Limit(c, 10, 20)
	if !ok {
		return
	}
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: cards})
}

func (h *Handler) Like(c *gin.Context) { h.swipe(c, DecisionLike) }
func (h *Handler) Pass(c *gin.Context) { h.swipe(c, DecisionPass) }

func (h *Handler) swipe(c *gin.Context, decision string) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), targetID, decision)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) LikesReceived(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 100)
	if !ok {
		return
	}
	cards, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		h.fail(c, "matching.likes_received", err)
		return
	}
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: cards})
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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileIncomplete):
		httpx.Fail(c, http.StatusConflict, "profile_incomplete", err.Error())
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Fail(c, http.StatusConflict, "already_swiped", err.Error())
	case errors.Is(err, ErrSelfSwipe):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
