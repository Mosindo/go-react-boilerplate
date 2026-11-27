package auth

import "github.com/gin-gonic/gin"

// Limits groups the rate limiters applied to public auth endpoints.
type Limits struct {
	Credentials gin.HandlerFunc // register, login
	Recovery    gin.HandlerFunc // forgot/reset password
	Refresh     gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limits) {
	r.POST("/auth/register", limits.Credentials, handler.Register)
	r.POST("/auth/login", limits.Credentials, handler.Login)
	r.POST("/auth/refresh", limits.Refresh, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", limits.Recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", limits.Recovery, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, limits.Credentials, handler.DeleteAccount)
}
