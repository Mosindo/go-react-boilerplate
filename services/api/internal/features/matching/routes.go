package matching

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Not a product limit: a flood guard against scripts (a human cannot swipe 300 times a minute).
	swipe := middleware.RateLimitByUser(300, time.Minute)
	browse := middleware.RateLimitByUser(120, time.Minute)

	r.GET("/discover", requireUser, browse, h.Discover)
	r.GET("/likes/received", requireUser, browse, h.LikesReceived)
	r.POST("/swipes", requireUser, swipe, h.Swipe)
}
