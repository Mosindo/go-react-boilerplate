package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts auth routes. publicLimit throttles the unauthenticated endpoints.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, publicLimit gin.HandlerFunc) {
	r.POST("/auth/register", publicLimit, handler.Register)
	r.POST("/auth/login", publicLimit, handler.Login)
	r.POST("/auth/refresh", publicLimit, handler.Refresh)
	r.POST("/auth/logout", publicLimit, handler.Logout)
	r.POST("/auth/forgot-password", publicLimit, handler.ForgotPassword)
	r.POST("/auth/reset-password", publicLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
