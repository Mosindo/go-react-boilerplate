package discover

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

func (h *Handler) List(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 10, 20)
	people, err := h.service.Next(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		if errors.Is(err, ErrNoProfile) {
			httpx.Fail(c, "discover.list", http.StatusConflict, err.Error(), err)
			return
		}
		httpx.Fail(c, "discover.list", http.StatusInternalServerError, "internal error", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": people})
}
