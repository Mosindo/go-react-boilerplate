package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, requireAdmin, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.ListBlocks)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)

	admin := r.Group("/admin", requireUser, requireAdmin)
	admin.GET("/reports", h.AdminListReports)
	admin.POST("/reports/:reportId/resolve", h.AdminResolveReport)
}
