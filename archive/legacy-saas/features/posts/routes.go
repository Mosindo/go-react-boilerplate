package posts

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/posts", requireUser, handler.List)
	r.POST("/posts", requireUser, handler.Create)
	r.GET("/posts/:postId", requireUser, handler.GetByID)
}
