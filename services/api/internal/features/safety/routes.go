package safety

import (
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	actions := ratelimit.ByUser("safety", ratelimit.New(60, time.Hour))

	r.GET("/blocks", requireUser, h.ListBlocks)
	r.POST("/blocks", requireUser, actions, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, actions, h.Report)
}
