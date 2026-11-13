package realtime

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler) {
	r.GET("/realtime", handler.Connect)
}
