package auth

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// Limits holds per-endpoint rate limiters.
type Limits struct {
	Credentials *ratelimit.Limiter // register/login/refresh
	Recovery    *ratelimit.Limiter // forgot/reset password
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limits) {
	r.POST("/auth/register", limits.Credentials.ByIP(), handler.Register)
	r.POST("/auth/login", limits.Credentials.ByIP(), handler.Login)
	r.POST("/auth/refresh", limits.Credentials.ByIP(), handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", limits.Recovery.ByIP(), handler.ForgotPassword)
	r.POST("/auth/reset-password", limits.Recovery.ByIP(), handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, limits.Recovery.ByUser(), handler.DeleteAccount)
}
