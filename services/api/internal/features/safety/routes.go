package safety

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func normalizeID(id string) string { return strings.ToLower(strings.TrimSpace(id)) }

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
