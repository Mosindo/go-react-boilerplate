package auth

import "github.com/gin-gonic/gin"

type RouteGuards struct {
	RequireUser   gin.HandlerFunc
	AuthLimit     gin.HandlerFunc // per-IP limiter for credential endpoints
	RecoveryLimit gin.HandlerFunc // stricter per-IP limiter for recovery
}

func RegisterRoutes(r gin.IRouter, h *Handler, g RouteGuards) {
	r.POST("/auth/register", g.AuthLimit, h.Register)
	r.POST("/auth/login", g.AuthLimit, h.Login)
	r.POST("/auth/refresh", g.AuthLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", g.RecoveryLimit, h.ForgotPassword)
	r.POST("/auth/password/reset", g.RecoveryLimit, h.ResetPassword)

	r.GET("/me", g.RequireUser, h.Me)
	r.PUT("/me/password", g.RequireUser, g.AuthLimit, h.ChangePassword)
	r.DELETE("/me", g.RequireUser, g.AuthLimit, h.DeleteAccount)
}
