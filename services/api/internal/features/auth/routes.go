package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts authentication and account endpoints. authLimit is a
// strict per-IP limiter protecting credential endpoints from brute force.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, authLimit gin.HandlerFunc) {
	r.POST("/auth/register", authLimit, handler.Register)
	r.POST("/auth/login", authLimit, handler.Login)
	r.POST("/auth/refresh", authLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", authLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", authLimit, handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, authLimit, handler.DeleteAccount)
	r.POST("/me/password", requireUser, authLimit, handler.ChangePassword)
	r.POST("/realtime/ticket", requireUser, handler.RealtimeTicket)
}
