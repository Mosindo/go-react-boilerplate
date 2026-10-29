package auth

import (
	"github.com/gin-gonic/gin"
)

// Limiters groups the rate limiters applied to the sensitive routes.
// Any field may be nil to disable that limit.
type Limiters struct {
	Auth    gin.HandlerFunc // login / register / refresh
	Recover gin.HandlerFunc // forgot / reset password
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits Limiters) {
	pass := func(c *gin.Context) { c.Next() }
	if limits.Auth == nil {
		limits.Auth = pass
	}
	if limits.Recover == nil {
		limits.Recover = pass
	}

	r.POST("/auth/register", limits.Auth, handler.Register)
	r.POST("/auth/login", limits.Auth, handler.Login)
	r.POST("/auth/refresh", limits.Auth, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", limits.Recover, handler.ForgotPassword)
	r.POST("/auth/reset-password", limits.Recover, handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.PUT("/me/password", requireUser, limits.Auth, handler.ChangePassword)
	r.DELETE("/me", requireUser, limits.Auth, handler.DeleteAccount)
}
