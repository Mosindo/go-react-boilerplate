package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the auth endpoints. `public` is a per-IP limiter for unauthenticated routes.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, public *middleware.Limiter) {
	limit := public.ByIP()
	r.POST("/auth/register", limit, h.Register)
	r.POST("/auth/login", limit, h.Login)
	r.POST("/auth/refresh", limit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/forgot-password", limit, h.ForgotPassword)
	r.POST("/auth/reset-password", limit, h.ResetPassword)
	r.PUT("/auth/password", requireUser, h.ChangePassword)
	r.GET("/me", requireUser, h.Me)
	r.DELETE("/me", requireUser, h.DeleteAccount)
}
