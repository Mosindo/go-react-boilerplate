package auth

import "github.com/gin-gonic/gin"

type RouteGuards struct {
	RequireUser gin.HandlerFunc
	// Credentials limits login/register/reset attempts per IP.
	Credentials gin.HandlerFunc
	// Refresh limits token refresh per IP.
	Refresh gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, handler *Handler, g RouteGuards) {
	r.POST("/auth/register", g.Credentials, handler.Register)
	r.POST("/auth/login", g.Credentials, handler.Login)
	r.POST("/auth/refresh", g.Refresh, handler.Refresh)
	r.POST("/auth/logout", g.Refresh, handler.Logout)
	r.POST("/auth/password/forgot", g.Credentials, handler.ForgotPassword)
	r.POST("/auth/password/reset", g.Credentials, handler.ResetPassword)
	r.GET("/me", g.RequireUser, handler.Me)
	r.DELETE("/me", g.RequireUser, g.Credentials, handler.DeleteAccount)
}
