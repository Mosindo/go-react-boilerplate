package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, handler.Feed)
}
