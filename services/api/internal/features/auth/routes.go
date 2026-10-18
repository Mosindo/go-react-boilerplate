package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// Limiters groups the rate limiters applied to the auth surface.
type Limiters struct {
	Auth    *middleware.RateLimiter // credential endpoints, per IP
	Recover *middleware.RateLimiter // password recovery, per IP
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limiters) {
	r.POST("/auth/register", limits.Auth.ByIP("register"), handler.Register)
	r.POST("/auth/login", limits.Auth.ByIP("login"), handler.Login)
	r.POST("/auth/refresh", limits.Auth.ByIP("refresh"), handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", limits.Recover.ByIP("forgot"), handler.ForgotPassword)
	r.POST("/auth/password/reset", limits.Recover.ByIP("reset"), handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, limits.Auth.ByUser("change-password"), handler.ChangePassword)
	r.DELETE("/me", requireUser, limits.Auth.ByUser("delete-account"), handler.DeleteAccount)
}
