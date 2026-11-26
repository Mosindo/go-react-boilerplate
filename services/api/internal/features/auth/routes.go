package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts auth endpoints. authLimit guards credential endpoints against brute force.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, authLimit gin.HandlerFunc) {
	r.POST("/auth/register", authLimit, h.Register)
	r.POST("/auth/login", authLimit, h.Login)
	r.POST("/auth/refresh", authLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", authLimit, h.ForgotPassword)
	r.POST("/auth/password/reset", authLimit, h.ResetPassword)

	r.GET("/me", requireUser, h.Me)
	r.PUT("/me/password", requireUser, authLimit, h.ChangePassword)
	r.DELETE("/me", requireUser, authLimit, h.DeleteAccount)
}
