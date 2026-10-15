package matching

import "github.com/gin-gonic/gin"

// swipeLimit is an anti-bot guard, deliberately far above what a person can do.
// The product has no like quota.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, swipeLimit, handler.Discover)
	r.GET("/profiles/:userId", requireUser, swipeLimit, handler.Profile)
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.DELETE("/matches/:userId", requireUser, handler.Unmatch)
}
