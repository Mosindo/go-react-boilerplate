package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, reportsPerHour int) {
	report := middleware.RateLimit(reportsPerHour, time.Hour, middleware.ByUser)
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, report, h.Report)
}
