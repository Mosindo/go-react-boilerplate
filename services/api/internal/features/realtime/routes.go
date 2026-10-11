package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.POST("/ws/ticket", requireUser, h.Ticket)
	r.GET("/ws", h.Connect)
}
