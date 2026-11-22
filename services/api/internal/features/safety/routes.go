package safety

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, reportLimiter *middleware.RateLimiter) {
	r.GET("/blocks", requireUser, h.ListBlocks)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, middleware.RateLimitByUser(reportLimiter, "reports"), h.Report)
}
