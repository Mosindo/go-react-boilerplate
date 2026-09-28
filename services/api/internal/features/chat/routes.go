package chat

import "github.com/gin-gonic/gin"

// RegisterRoutes wires the conversation endpoints. sendMW (per-user rate limit)
// applies to sending only.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, sendMW ...gin.HandlerFunc) {
	g := r.Group("/conversations", requireUser)
	g.GET("/:id/messages", h.Messages)
	g.POST("/:id/messages", append(append([]gin.HandlerFunc{}, sendMW...), h.Send)...)
	g.POST("/:id/read", h.MarkRead)
}
