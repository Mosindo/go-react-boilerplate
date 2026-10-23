package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.POST("/swipes", h.Swipe)
	g.GET("/matches", h.List)
	g.DELETE("/matches/:id", h.Unmatch)
	g.POST("/matches/:id/hide", h.Hide)
}
