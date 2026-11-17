package auth

import "github.com/gin-gonic/gin"

type RouteLimits struct {
	Login    gin.HandlerFunc
	Register gin.HandlerFunc
	Reset    gin.HandlerFunc
	Refresh  gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits RouteLimits) {
	r.POST("/auth/register", limits.Register, handler.Register)
	r.POST("/auth/login", limits.Login, handler.Login)
	r.POST("/auth/refresh", limits.Refresh, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", limits.Reset, handler.ForgotPassword)
	r.POST("/auth/password/reset", limits.Reset, handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, limits.Login, handler.ChangePassword)
	r.DELETE("/me", requireUser, limits.Login, handler.DeleteAccount)
}
