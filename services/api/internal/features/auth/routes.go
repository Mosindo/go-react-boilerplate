package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts public auth routes (limited by authLimit) and
// authenticated account routes.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, authLimit gin.HandlerFunc) {
	pub := r.Group("/auth", authLimit)
	pub.POST("/register", h.Register)
	pub.POST("/login", h.Login)
	pub.POST("/refresh", h.Refresh)
	pub.POST("/logout", h.Logout)
	pub.POST("/password/forgot", h.ForgotPassword)
	pub.POST("/password/reset", h.ResetPassword)

	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, authLimit, h.ChangePassword)
	r.POST("/me/delete", requireUser, authLimit, h.DeleteAccount)
}
