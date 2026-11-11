package auth

import "github.com/gin-gonic/gin"

type RouteGuards struct {
	RequireUser gin.HandlerFunc
	// Strict is applied to credential endpoints (login, register, reset).
	Strict gin.HandlerFunc
	// Standard is applied to authenticated account endpoints.
	Standard gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, h *Handler, g RouteGuards) {
	r.POST("/auth/register", g.Strict, h.Register)
	r.POST("/auth/login", g.Strict, h.Login)
	r.POST("/auth/refresh", g.Standard, h.Refresh)
	r.POST("/auth/logout", g.Standard, h.Logout)
	r.POST("/auth/password/forgot", g.Strict, h.ForgotPassword)
	r.POST("/auth/password/reset", g.Strict, h.ResetPassword)
	r.GET("/me", g.RequireUser, h.Me)
	r.DELETE("/me", g.RequireUser, g.Strict, h.DeleteAccount)
}
