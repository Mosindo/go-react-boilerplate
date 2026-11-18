package matching

import "github.com/gin-gonic/gin"

// swipeLimit is an anti-automation guard (generous, never a product quota):
// likes are unlimited for humans.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.GET("/matches", requireUser, handler.List)
	r.DELETE("/matches/:userId", requireUser, handler.Unmatch)
}
