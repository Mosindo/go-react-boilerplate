package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes wires auth endpoints. authLimit throttles credential
// endpoints per IP to slow down brute force and enumeration.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, authLimit gin.HandlerFunc) {
	r.POST("/auth/register", authLimit, h.Register)
	r.POST("/auth/login", authLimit, h.Login)
	r.POST("/auth/refresh", authLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", authLimit, h.ForgotPassword)
	r.POST("/auth/password/reset", authLimit, h.ResetPassword)

	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, authLimit, h.ChangePassword)
	r.DELETE("/me", requireUser, authLimit, h.DeleteAccount)
}
