package safety

import "github.com/gin-gonic/gin"

// RegisterRoutes wires blocks and reports. reportMW is the per-user report rate limit.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, reportMW ...gin.HandlerFunc) {
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.GET("/blocks", requireUser, h.List)
	r.POST("/reports", append(append([]gin.HandlerFunc{requireUser}, reportMW...), h.Report)...)
}
