package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	res, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 20))
	if err != nil {
		httpx.Fail(c, "discovery.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "discovery.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ResetPasses(c *gin.Context) {
	if err := h.service.ResetPasses(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "discovery.reset_passes", err)
		return
	}
	c.Status(http.StatusNoContent)
}
