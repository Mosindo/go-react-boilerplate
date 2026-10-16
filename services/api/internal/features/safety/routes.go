package safety

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	limit := middleware.NewLimiter(60, time.Minute).ByUser()
	reportLimit := middleware.NewLimiter(10, time.Hour).ByUser()

	r.GET("/blocks", requireUser, limit, h.Blocked)
	r.POST("/blocks", requireUser, limit, h.Block)
	r.DELETE("/blocks/:userId", requireUser, limit, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
