package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("/conversations", requireUser)
	g.GET("", h.ListConversations)
	g.GET("/:id", h.GetConversation)
	g.GET("/:id/messages", h.ListMessages)
	g.POST("/:id/messages", middleware.RateLimitByUser(90, 20), h.Send)
	g.POST("/:id/read", h.MarkRead)
	g.DELETE("/:id", h.Hide)
}
