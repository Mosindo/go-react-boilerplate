package matches

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, err := httpx.ParseLimit(c, DefaultLimit, MaxLimit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	page, err := h.service.List(c.Request.Context(), httpx.UserID(c), c.Query("cursor"), limit)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), httpx.UserID(c), c.Param("matchId")); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
