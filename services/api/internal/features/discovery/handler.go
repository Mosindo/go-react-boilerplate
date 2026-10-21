package discovery

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Next(c *gin.Context) {
	page, err := h.service.Next(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 10, 20))
	if err != nil {
		httpx.Fail(c, "discovery.next", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, page)
}
