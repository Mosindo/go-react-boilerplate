package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discovery", requireUser, h.Feed)
	r.POST("/discovery/swipes", requireUser, swipeLimit, h.Swipe)
}
