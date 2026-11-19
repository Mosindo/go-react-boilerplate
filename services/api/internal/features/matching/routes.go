package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/discovery", h.Discover)
	g.POST("/discovery/:userId/like", swipeLimit, h.Like)
	g.POST("/discovery/:userId/pass", swipeLimit, h.Pass)
	g.GET("/likes/received", h.LikesReceived)
	g.DELETE("/matches/:matchId", h.Unmatch)
}
