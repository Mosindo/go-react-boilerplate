package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Discover(c *gin.Context) {
	limit, err := httpx.LimitWith(c, 10, 20)
	if err != nil {
		httpx.Fail(c, "discover", err)
		return
	}
	list, err := h.service.Discover(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		httpx.Fail(c, "discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}
