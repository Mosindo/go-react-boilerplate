package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the auth routes. publicLimit protects the unauthenticated endpoints against
// brute force and abuse; requireUser guards the account endpoints.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, publicLimit gin.HandlerFunc) {
	r.POST("/auth/register", publicLimit, handler.Register)
	r.POST("/auth/login", publicLimit, handler.Login)
	r.POST("/auth/refresh", publicLimit, handler.Refresh)
	r.POST("/auth/logout", publicLimit, handler.Logout)
	r.POST("/auth/password/forgot", publicLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", publicLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, publicLimit, handler.ChangePassword)
	r.DELETE("/me", requireUser, publicLimit, handler.DeleteAccount)
}
