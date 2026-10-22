package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeLimiter *middleware.RateLimiter) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/discover/swipes", requireUser, swipeLimiter.ByUser(), h.Swipe)
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
