package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the auth endpoints. The limiters are injected so the
// server decides the budgets: public protects credential endpoints, recovery
// protects the password-reset flow.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, publicLimit, recoveryLimit gin.HandlerFunc) {
	public := r.Group("/auth", publicLimit)
	public.POST("/register", handler.Register)
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.Refresh)
	public.POST("/logout", handler.Logout)

	recovery := r.Group("/auth/password", recoveryLimit)
	recovery.POST("/forgot", handler.ForgotPassword)
	recovery.POST("/reset", handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/delete", requireUser, handler.DeleteAccount)
}
