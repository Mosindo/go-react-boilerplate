package matches

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, messageLimit gin.HandlerFunc) {
	g := r.Group("/matches", requireUser)
	g.GET("", h.List)
	g.GET("/:matchId", h.Get)
	g.DELETE("/:matchId", h.Unmatch)
	g.GET("/:matchId/messages", h.Messages)
	g.POST("/:matchId/messages", messageLimit, h.Send)
	g.POST("/:matchId/read", h.MarkRead)
}
