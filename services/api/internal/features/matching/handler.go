package matching

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Discover(c *gin.Context) {
	cards, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), DefaultDiscoveryLimit, MaxDiscoveryLimit))
	if err != nil {
		httpx.Fail(c, "matching.discover", err)
		return
	}
	c.JSON(http.StatusOK, DiscoveryResponse{Profiles: cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, "matching.swipe", &req) {
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.TargetUserID), req.Decision)
	if err != nil {
		httpx.Fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	matchID, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), matchID); err != nil {
		httpx.Fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
