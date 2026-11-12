package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/users", requireUser, h.List)
	r.GET("/users/:userId", requireUser, h.GetByID)
}
