package matching

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Swiping is never capped for product reasons; this only stops scripted floods.
	swipeLimit := middleware.NewLimiter(240, time.Minute).ByUser()
	readLimit := middleware.NewLimiter(120, time.Minute).ByUser()

	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.GET("/matches", requireUser, readLimit, h.Matches)
	r.DELETE("/matches/:matchId", requireUser, readLimit, h.Unmatch)
	r.GET("/likes/received", requireUser, readLimit, h.LikesReceived)
}
