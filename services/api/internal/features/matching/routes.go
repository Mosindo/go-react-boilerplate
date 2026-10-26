package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/discover/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches/:id", requireUser, h.GetMatch)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
