package discovery

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Generous: this only stops scripted scraping/bots, not real use (no artificial like quota).
	swipes := middleware.Limit(300, time.Minute, middleware.ByUser)
	browse := middleware.Limit(120, time.Minute, middleware.ByUser)

	r.GET("/discover", requireUser, browse, h.Discover)
	r.GET("/users/:userId/profile", requireUser, browse, h.Profile)
	r.POST("/swipes", requireUser, swipes, h.Swipe)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
