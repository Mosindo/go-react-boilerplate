package chat

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	send := ratelimit.ByUser(ratelimit.New(30, 60))
	r.GET("/matches/:id/messages", requireUser, h.List)
	r.POST("/matches/:id/messages", requireUser, send, h.Send)
	r.POST("/matches/:id/read", requireUser, h.MarkRead)
	r.DELETE("/matches/:id/messages", requireUser, h.Clear)
}
