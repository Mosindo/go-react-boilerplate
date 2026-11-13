package auth

import "github.com/gin-gonic/gin"

// Limits groups the per-route rate limit middlewares.
type Limits struct {
	Register gin.HandlerFunc
	Login    gin.HandlerFunc
	Recovery gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limits) {
	r.POST("/auth/register", limits.Register, handler.Register)
	r.POST("/auth/login", limits.Login, handler.Login)
	r.POST("/auth/refresh", limits.Login, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", limits.Recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", limits.Recovery, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, limits.Login, handler.DeleteAccount)
}
