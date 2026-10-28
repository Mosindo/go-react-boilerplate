package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	blocks := middleware.RateLimitByUser(middleware.NewLimiter(60, time.Hour))
	reports := middleware.RateLimitByUser(middleware.NewLimiter(10, time.Hour))

	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, blocks, h.Block)
	r.DELETE("/blocks/:userId", requireUser, blocks, h.Unblock)
	r.POST("/reports", requireUser, reports, h.Report)
}
