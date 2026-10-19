package safety

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/blocks", h.ListBlocks)
	g.POST("/blocks", middleware.RateLimitByUser(30, 10), h.Block)
	g.DELETE("/blocks/:userId", h.Unblock)
	g.POST("/reports", middleware.RateLimitByUser(20, 10), h.Report)
}
