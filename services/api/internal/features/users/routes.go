package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
