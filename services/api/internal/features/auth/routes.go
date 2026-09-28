package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes wires /auth/* (public, wrapped by publicMW: per-IP rate limit)
// and the authenticated account routes. GET /me lives in the profiles feature.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, publicMW ...gin.HandlerFunc) {
	pub := r.Group("/auth", publicMW...)
	pub.POST("/register", h.Register)
	pub.POST("/login", h.Login)
	pub.POST("/refresh", h.Refresh)
	pub.POST("/logout", h.Logout)
	pub.POST("/forgot-password", h.ForgotPassword)
	pub.POST("/reset-password", h.ResetPassword)

	r.POST("/me/password", requireUser, h.ChangePassword)
	r.DELETE("/me", requireUser, h.DeleteAccount)
}
