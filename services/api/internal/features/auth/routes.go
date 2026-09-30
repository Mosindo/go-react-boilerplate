package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, authLimit gin.HandlerFunc) {
	r.POST("/auth/register", authLimit, handler.Register)
	r.POST("/auth/login", authLimit, handler.Login)
	r.POST("/auth/refresh", authLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", authLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", authLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, authLimit, handler.DeleteAccount)
}
