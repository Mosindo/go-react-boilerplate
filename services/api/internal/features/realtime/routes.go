package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, connectRL gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, h.Ticket)
	r.GET("/realtime", connectRL, h.Connect)
}
