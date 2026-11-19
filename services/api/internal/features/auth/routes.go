package auth

import "github.com/gin-gonic/gin"

type RouteGuards struct {
	RequireUser gin.HandlerFunc
	// Credential endpoints (register, login, refresh) — limited per IP.
	CredentialLimit gin.HandlerFunc
	// Recovery endpoints — stricter per-IP limit.
	RecoveryLimit gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, h *Handler, g RouteGuards) {
	r.POST("/auth/register", g.CredentialLimit, h.Register)
	r.POST("/auth/login", g.CredentialLimit, h.Login)
	r.POST("/auth/refresh", g.CredentialLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", g.RecoveryLimit, h.ForgotPassword)
	r.POST("/auth/password/reset", g.RecoveryLimit, h.ResetPassword)
	r.GET("/me", g.RequireUser, h.Me)
	r.DELETE("/me", g.RequireUser, g.RecoveryLimit, h.DeleteAccount)
}
