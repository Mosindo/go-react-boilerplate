package matching

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts discovery, swipes and matches. swipeLimit is an abuse guard, not a product quota.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
