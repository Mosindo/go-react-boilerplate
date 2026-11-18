package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, ticketLimit gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, ticketLimit, handler.Ticket)
	r.GET("/realtime/ws", handler.Connect)
}
