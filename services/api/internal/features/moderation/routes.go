package moderation

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	reportLimit := middleware.NewLimiter(20, time.Hour).ByUser()
	blockLimit := middleware.NewLimiter(120, time.Hour).ByUser()
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, blockLimit, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
