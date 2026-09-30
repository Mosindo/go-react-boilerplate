package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.DELETE("/matches/:matchId", requireUser, handler.Unmatch)
}
