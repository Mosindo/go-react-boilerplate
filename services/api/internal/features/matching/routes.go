package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/likes/received", requireUser, h.LikesReceived)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
