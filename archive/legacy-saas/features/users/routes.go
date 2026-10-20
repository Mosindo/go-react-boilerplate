package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/users", requireUser, handler.List)
	r.GET("/users/:userId", requireUser, handler.GetByID)
}
