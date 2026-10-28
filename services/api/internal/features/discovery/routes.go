package discovery

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Abuse guard, not a product limit: far above what a person can do by hand.
	swipes := middleware.RateLimitByUser(middleware.NewLimiter(120, time.Minute))
	discover := middleware.RateLimitByUser(middleware.NewLimiter(240, time.Minute))
	reset := middleware.RateLimitByUser(middleware.NewLimiter(10, time.Hour))

	r.GET("/discover", requireUser, discover, h.Discover)
	r.POST("/swipes", requireUser, swipes, h.Swipe)
	r.POST("/swipes/reset-passes", requireUser, reset, h.ResetPasses)
}
