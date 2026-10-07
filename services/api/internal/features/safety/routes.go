package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.Blocks)
	r.POST("/blocks", requireUser, middleware.Limit(60, time.Hour, middleware.ByUser), h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, middleware.Limit(20, time.Hour, middleware.ByUser), h.Report)
}
