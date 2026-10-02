package auth

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// 20 attempts burst per IP, then 10/min: blocks credential stuffing, invisible to humans.
	limit := ratelimit.ByIP(ratelimit.New(20, 10))
	r.POST("/auth/register", limit, h.Register)
	r.POST("/auth/login", limit, h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/forgot-password", limit, h.ForgotPassword)
	r.POST("/auth/reset-password", limit, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, h.ChangePassword)
	r.DELETE("/me", requireUser, h.DeleteAccount)
}
