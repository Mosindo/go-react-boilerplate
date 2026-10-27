package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipes *middleware.RateLimiter) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, middleware.RateLimit(swipes), h.Swipe)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
	r.GET("/users/:id/profile", requireUser, h.Profile)
}
