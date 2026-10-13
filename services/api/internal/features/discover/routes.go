package discover

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	r.GET("/discover", requireUser, limiter.ByUser("discover", 120, time.Minute), h.Next)
}
