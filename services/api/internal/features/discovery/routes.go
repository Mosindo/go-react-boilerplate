package discovery

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Generous anti-abuse ceiling (not a product limit: likes are unlimited for normal use).
	swipeLimit := middleware.NewLimiter(600, time.Hour).ByUser()
	r.GET("/discover", requireUser, h.Feed)
	r.POST("/discover/swipes", requireUser, swipeLimit, h.Swipe)
	r.DELETE("/discover/swipes/:userId", requireUser, h.Unswipe)
	r.GET("/profiles/:userId", requireUser, h.Profile)
	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
