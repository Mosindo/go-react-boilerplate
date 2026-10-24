package matching

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipes *ratelimit.Limiter) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipes.ByUser(), h.Swipe)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
