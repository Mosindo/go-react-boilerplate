package matches

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Abuse guard only: 120 swipes burst, then 60/min. Likes are never capped per day.
	swipes := ratelimit.ByUser(ratelimit.New(120, 60))
	r.POST("/swipes", requireUser, swipes, h.Swipe)
	r.GET("/matches", requireUser, h.List)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
