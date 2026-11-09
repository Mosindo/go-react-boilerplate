package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/discover/reset", requireUser, h.ResetPasses)
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
