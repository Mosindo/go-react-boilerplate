package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, ticketLimit gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, ticketLimit, h.Ticket)
	r.GET("/realtime", h.Connect)
}
