package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeRL gin.HandlerFunc) {
	r.GET("/discovery", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeRL, h.Swipe)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
