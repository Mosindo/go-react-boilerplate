package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts auth routes. authLimit protects unauthenticated
// endpoints against brute force and abuse.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, authLimit gin.HandlerFunc) {
	r.POST("/auth/register", authLimit, handler.Register)
	r.POST("/auth/login", authLimit, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", authLimit, handler.ForgotPassword)
	r.POST("/auth/reset-password", authLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
