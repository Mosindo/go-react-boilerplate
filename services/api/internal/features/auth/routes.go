package auth

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts auth endpoints. credentialLimit throttles unauthenticated, brute-forceable routes.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, credentialLimit gin.HandlerFunc) {
	r.POST("/auth/register", credentialLimit, handler.Register)
	r.POST("/auth/login", credentialLimit, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", credentialLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", credentialLimit, handler.ResetPassword)
	r.POST("/auth/password/change", requireUser, credentialLimit, handler.ChangePassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, credentialLimit, handler.DeleteAccount)
}
