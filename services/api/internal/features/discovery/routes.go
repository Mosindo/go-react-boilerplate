package discovery

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeLimiter *middleware.Limiter) {
	r.GET("/discover", requireUser, h.Feed)
	r.POST("/swipes", requireUser, middleware.RateLimitByUser(swipeLimiter), h.Swipe)
	r.GET("/likes/received", requireUser, h.LikesReceived)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
