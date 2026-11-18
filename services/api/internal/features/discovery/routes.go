package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discovery", requireUser, handler.Feed)
	r.GET("/profiles/:userId", requireUser, handler.Profile)
}
