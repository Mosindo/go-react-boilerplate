package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/discover", h.Deck)
	g.POST("/swipes", swipeLimit, h.Swipe)
	g.GET("/likes/received", h.LikesReceived)
	g.GET("/matches", h.ListMatches)
	g.DELETE("/matches/:matchId", h.Unmatch)
}
