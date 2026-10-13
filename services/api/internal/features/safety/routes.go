package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	r.GET("/blocks", requireUser, h.Blocked)
	r.POST("/blocks", requireUser, limiter.ByUser("block", 60, time.Hour), h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, limiter.ByUser("report", 10, time.Hour), h.Report)
}
