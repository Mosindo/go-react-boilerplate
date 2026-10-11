package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	profiles, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 20))
	if err != nil {
		httpx.Respond(c, "discovery.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (h *Handler) LikesReceived(c *gin.Context) {
	profiles, err := h.service.LikesReceived(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 20, 50))
	if err != nil {
		httpx.Respond(c, "discovery.likes", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId (uuid) and action (like|pass) are required")
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req.UserID, req.Action)
	if err != nil {
		httpx.Respond(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "matchId")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), id, httpx.UserID(c)); err != nil {
		httpx.Respond(c, "discovery.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
