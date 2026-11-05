package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Feed)
	r.GET("/profiles/:id", requireUser, h.Profile)
	r.POST("/swipes", requireUser, h.Swipe)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
