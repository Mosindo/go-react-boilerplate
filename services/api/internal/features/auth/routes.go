package auth

import "github.com/gin-gonic/gin"

// Guards groups the middleware applied to auth routes.
type Guards struct {
	RequireUser   gin.HandlerFunc
	CredentialsRL gin.HandlerFunc
	RecoveryRL    gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, h *Handler, g Guards) {
	r.POST("/auth/register", g.CredentialsRL, h.Register)
	r.POST("/auth/login", g.CredentialsRL, h.Login)
	r.POST("/auth/refresh", g.CredentialsRL, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", g.RecoveryRL, h.ForgotPassword)
	r.POST("/auth/password/reset", g.RecoveryRL, h.ResetPassword)
	r.GET("/me", g.RequireUser, h.Me)
	r.DELETE("/me", g.RequireUser, g.CredentialsRL, h.DeleteAccount)
}
