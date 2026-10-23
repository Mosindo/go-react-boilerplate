package chat

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts chat routes. sendLimit throttles message creation per user;
// middlewares (auth first) apply to every route.
func RegisterRoutes(r gin.IRouter, h *Handler, sendLimit gin.HandlerFunc, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/matches/:id/messages", h.List)
	g.POST("/matches/:id/messages", sendLimit, h.Send)
	g.POST("/matches/:id/read", h.MarkRead)
}
