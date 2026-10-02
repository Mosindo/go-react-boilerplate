package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.GET("/profiles/:id", requireUser, h.Profile)
}
