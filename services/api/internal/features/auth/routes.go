package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the auth endpoints. authLimiter protects the unauthenticated
// credential endpoints against brute force and abuse.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, authLimiter gin.HandlerFunc) {
	r.POST("/auth/register", authLimiter, handler.Register)
	r.POST("/auth/login", authLimiter, handler.Login)
	r.POST("/auth/refresh", authLimiter, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password-reset/request", authLimiter, handler.RequestPasswordReset)
	r.POST("/auth/password-reset/confirm", authLimiter, handler.ConfirmPasswordReset)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
