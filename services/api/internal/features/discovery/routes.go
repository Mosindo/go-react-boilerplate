package discovery

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts discovery routes. `swipeLimit` is an anti-abuse throttle,
// not a product limit: it is deliberately generous.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/discover", h.Deck)
	g.GET("/likes/received", h.LikesReceived)
	g.POST("/swipes", swipeLimit, h.Swipe)
}
