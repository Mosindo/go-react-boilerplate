package discovery

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipesPerMinute int) {
	swipe := middleware.RateLimit(swipesPerMinute, time.Minute, middleware.ByUser)
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipe, h.Swipe)
	r.GET("/likes/received", requireUser, h.LikesReceived)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
