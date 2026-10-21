package matches

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if !httpx.Bind(c, &req) {
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "matches.swipe", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ResetPasses(c *gin.Context) {
	if err := h.service.ResetPasses(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "matches.reset_passes", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("before"), httpx.Limit(c, 30, 50))
	if err != nil {
		httpx.Fail(c, "matches.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "matches.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
