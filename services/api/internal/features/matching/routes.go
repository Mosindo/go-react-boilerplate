package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimiter gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeLimiter, h.Swipe)
}
