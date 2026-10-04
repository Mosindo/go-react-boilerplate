package discovery

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeLimit *middleware.RateLimiter) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeLimit.ByUser(), h.Swipe)
}
