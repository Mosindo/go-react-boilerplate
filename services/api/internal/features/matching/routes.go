package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeLimiter *middleware.RateLimiter) {
	r.GET("/discovery", requireUser, h.Discover)
	r.DELETE("/discovery/passes", requireUser, h.ResetPasses)
	r.POST("/swipes", requireUser, middleware.RateLimitByUser(swipeLimiter, "swipes"), h.Swipe)
	r.GET("/likes/received", requireUser, h.LikesReceived)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:userId", requireUser, h.Unmatch)
}
