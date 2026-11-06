package auth

import "github.com/gin-gonic/gin"

// Limiters groups the rate limits applied to unauthenticated endpoints.
type Limiters struct {
	Auth    gin.HandlerFunc // login / register / refresh
	Recover gin.HandlerFunc // password recovery
	Account gin.HandlerFunc // sensitive authenticated actions
}

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, l Limiters) {
	r.POST("/auth/register", l.Auth, h.Register)
	r.POST("/auth/login", l.Auth, h.Login)
	r.POST("/auth/refresh", l.Auth, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", l.Recover, h.ForgotPassword)
	r.POST("/auth/password/reset", l.Recover, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
	r.PUT("/me/password", requireUser, l.Account, h.ChangePassword)
	r.DELETE("/me", requireUser, l.Account, h.DeleteAccount)
}
