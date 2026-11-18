package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes wires the auth endpoints. anonLimit throttles unauthenticated
// endpoints per client IP to slow down credential stuffing and code guessing.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, anonLimit gin.HandlerFunc) {
	r.POST("/auth/register", anonLimit, handler.Register)
	r.POST("/auth/login", anonLimit, handler.Login)
	r.POST("/auth/refresh", anonLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", anonLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", anonLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.PUT("/me/password", requireUser, handler.ChangePassword)
}
