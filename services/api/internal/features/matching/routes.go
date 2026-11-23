package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
