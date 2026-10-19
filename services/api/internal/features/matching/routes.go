package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/discover", middleware.RateLimitByUser(120, 30), h.Discover)
	// Abuse guard only: generous enough that a human can never hit it.
	g.POST("/swipes", middleware.RateLimitByUser(300, 60), h.Swipe)
	g.GET("/matches", h.ListMatches)
	g.DELETE("/matches/:id", h.Unmatch)
	g.GET("/likes/received", h.ReceivedLikes)
}
