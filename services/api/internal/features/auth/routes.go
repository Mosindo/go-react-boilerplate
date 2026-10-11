package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// Limits configures per-IP request budgets for the unauthenticated endpoints.
type Limits struct {
	AuthPerMinute  int
	ResetPerHour   int
	SensitivePerHr int
}

func DefaultLimits() Limits { return Limits{AuthPerMinute: 10, ResetPerHour: 5, SensitivePerHr: 10} }

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, l Limits) {
	authLimit := middleware.RateLimit(l.AuthPerMinute, time.Minute, middleware.ByIP)
	resetLimit := middleware.RateLimit(l.ResetPerHour, time.Hour, middleware.ByIP)
	sensitiveLimit := middleware.RateLimit(l.SensitivePerHr, time.Hour, middleware.ByUser)

	r.POST("/auth/register", authLimit, h.Register)
	r.POST("/auth/login", authLimit, h.Login)
	r.POST("/auth/refresh", authLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/forgot-password", resetLimit, h.ForgotPassword)
	r.POST("/auth/reset-password", resetLimit, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, sensitiveLimit, h.ChangePassword)
	r.DELETE("/me", requireUser, sensitiveLimit, h.DeleteAccount)
}
