package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, swipeLimit gin.HandlerFunc) {
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.POST("/swipes/undo", requireUser, swipeLimit, handler.Undo)
	r.DELETE("/matches/:matchId", requireUser, handler.Unmatch)
}
