package matching

import (
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Swipes are not capped for real users (the product is free and unlimited);
	// this ceiling only stops scripted abuse.
	swipes := ratelimit.ByUser("swipe", ratelimit.New(240, time.Minute))
	discover := ratelimit.ByUser("discover", ratelimit.New(120, time.Minute))

	r.GET("/discover", requireUser, discover, h.Discover)
	r.POST("/swipes", requireUser, swipes, h.Swipe)
	r.GET("/likes/received", requireUser, h.LikesReceived)
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
