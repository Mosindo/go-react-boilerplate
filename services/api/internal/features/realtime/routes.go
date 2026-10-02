package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, h.Ticket)
	r.GET("/realtime/ws", h.Connect)
}
