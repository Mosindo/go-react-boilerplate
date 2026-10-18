package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipes *middleware.RateLimiter) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipes.ByUser("swipe"), h.Swipe)
	r.DELETE("/swipes/passes", requireUser, h.ResetPasses)
	r.GET("/profiles/:userId", requireUser, h.Profile)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
