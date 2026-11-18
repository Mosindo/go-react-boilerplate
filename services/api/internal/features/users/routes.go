package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, sensitiveLimit gin.HandlerFunc) {
	r.DELETE("/me", requireUser, sensitiveLimit, handler.DeleteAccount)
}
