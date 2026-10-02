package safety

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	reports := ratelimit.ByUser(ratelimit.New(10, 6))
	r.GET("/blocks", requireUser, h.ListBlocked)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reports, h.Report)
}
