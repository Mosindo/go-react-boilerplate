package safety

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, reportLimiter *middleware.Limiter) {
	r.POST("/blocks", requireUser, handler.Block)
	r.GET("/blocks", requireUser, handler.ListBlocks)
	r.DELETE("/blocks/:userId", requireUser, handler.Unblock)
	r.POST("/reports", requireUser, middleware.RateLimitByUser(reportLimiter), handler.Report)
}
