package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts auth routes. `strict` throttles credential endpoints.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, strict gin.HandlerFunc) {
	r.POST("/auth/register", strict, h.Register)
	r.POST("/auth/login", strict, h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", strict, h.ForgotPassword)
	r.POST("/auth/password/reset", strict, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
}
