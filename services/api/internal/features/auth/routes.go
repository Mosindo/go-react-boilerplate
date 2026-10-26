package auth

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts auth endpoints. credentialLimit protects endpoints that
// accept secrets (login, register, reset) against brute force.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, credentialLimit gin.HandlerFunc) {
	r.POST("/auth/register", credentialLimit, handler.Register)
	r.POST("/auth/login", credentialLimit, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", credentialLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", credentialLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, credentialLimit, handler.ChangePassword)
	r.DELETE("/me", requireUser, credentialLimit, handler.DeleteAccount)
}
