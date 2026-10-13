package matches

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	r.POST("/likes", requireUser, limiter.ByUser("swipe", 200, time.Minute), h.Swipe)
	r.GET("/matches", requireUser, h.List)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
