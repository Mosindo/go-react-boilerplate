package discovery

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Generous enough for fast swiping, tight enough to stop scripted scraping.
	swipeLimit := middleware.RateLimitByUser(middleware.NewLimiter(600, time.Hour))
	discoverLimit := middleware.RateLimitByUser(middleware.NewLimiter(240, time.Hour))
	r.GET("/discover", requireUser, discoverLimit, h.Discover)
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
