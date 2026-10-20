package discovery

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, swipeLimiter *middleware.Limiter) {
	r.GET("/discover", requireUser, handler.Discover)
	r.POST("/swipes", requireUser, middleware.RateLimitByUser(swipeLimiter), handler.Swipe)
	r.GET("/matches", requireUser, handler.ListMatches)
	r.DELETE("/matches/:id", requireUser, handler.Unmatch)
}
