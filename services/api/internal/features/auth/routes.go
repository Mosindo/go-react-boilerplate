package auth

import "github.com/gin-gonic/gin"

type RouteGuards struct {
	RequireUser gin.HandlerFunc
	// Credentials guards login/register/reset endpoints (per IP).
	Credentials gin.HandlerFunc
	// Sensitive guards authenticated, password-checking endpoints (per user).
	Sensitive gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, handler *Handler, guards RouteGuards) {
	r.POST("/auth/register", guards.Credentials, handler.Register)
	r.POST("/auth/login", guards.Credentials, handler.Login)
	r.POST("/auth/refresh", guards.Credentials, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", guards.Credentials, handler.ForgotPassword)
	r.POST("/auth/password/reset", guards.Credentials, handler.ResetPassword)

	r.GET("/me", guards.RequireUser, handler.Me)
	r.POST("/me/password", guards.RequireUser, guards.Sensitive, handler.ChangePassword)
	r.DELETE("/me", guards.RequireUser, guards.Sensitive, handler.DeleteAccount)
}
