package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	reportLimit := middleware.RateLimitByUser(middleware.NewLimiter(10, time.Hour))
	r.GET("/blocks", requireUser, h.ListBlocked)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:id", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
