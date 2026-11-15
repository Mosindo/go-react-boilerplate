package auth

import "github.com/gin-gonic/gin"

// Limits groups the abuse-protection middleware applied to sensitive routes.
type Limits struct {
	Credentials gin.HandlerFunc // login, register, reset (per IP)
	Recovery    gin.HandlerFunc // forgot-password e-mails (per IP)
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limits) {
	r.POST("/auth/register", limits.Credentials, handler.Register)
	r.POST("/auth/login", limits.Credentials, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", limits.Recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", limits.Credentials, handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.PUT("/me/password", requireUser, limits.Credentials, handler.ChangePassword)
	r.DELETE("/me", requireUser, limits.Credentials, handler.DeleteAccount)
}
