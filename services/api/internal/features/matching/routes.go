package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discovery", requireUser, handler.Discover)
	r.POST("/discovery/swipes", requireUser, swipeLimit, handler.Swipe)
	r.GET("/matches", requireUser, handler.ListMatches)
	r.GET("/matches/:matchId", requireUser, handler.GetMatch)
	r.DELETE("/matches/:matchId", requireUser, handler.Unmatch)
}
