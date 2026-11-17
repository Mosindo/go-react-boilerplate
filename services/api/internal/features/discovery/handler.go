package discovery

import (
	"net/http"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Feed(c *gin.Context) {
	limit, ok := httpx.Limit(c, DefaultLimit, MaxLimit)
	if !ok {
		return
	}
	feed, err := h.service.Feed(c.Request.Context(), httpx.UserID(c), limit)
	if err != nil {
		if status, code, msg, ok := profiles.MapError(err); ok {
			httpx.Fail(c, status, code, msg)
			return
		}
		httpx.FailInternal(c, "discovery.feed", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, feed)
}
