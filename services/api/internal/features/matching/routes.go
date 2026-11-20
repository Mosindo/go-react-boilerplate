package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discovery", requireUser, handler.Discover)
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.GET("/likes/received", requireUser, handler.LikesReceived)
	r.DELETE("/matches/:matchId", requireUser, handler.Unmatch)
}
