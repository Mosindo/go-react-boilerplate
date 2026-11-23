package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, limit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.ListBlocks)
	r.POST("/blocks", requireUser, limit, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, limit, h.Report)
}
