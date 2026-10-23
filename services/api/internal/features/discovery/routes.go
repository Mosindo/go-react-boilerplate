package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/discover", h.Discover)
	g.GET("/profiles/:userId", h.Profile)
}
