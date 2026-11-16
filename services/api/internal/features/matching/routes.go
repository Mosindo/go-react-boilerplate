package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches", requireUser, h.List)
	r.GET("/matches/:matchId", requireUser, h.Get)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
