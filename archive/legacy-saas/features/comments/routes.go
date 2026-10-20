package comments

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/posts/:postId/comments", requireUser, handler.ListByPost)
	r.POST("/posts/:postId/comments", requireUser, handler.Create)
}
