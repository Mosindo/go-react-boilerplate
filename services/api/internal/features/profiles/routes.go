package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	me := r.Group("/me", requireUser)
	me.GET("/profile", h.Mine)
	me.PUT("/profile", h.SaveProfile)
	me.PUT("/preferences", h.SavePreferences)
	me.PUT("/privacy", h.SetPrivacy)
	r.GET("/profiles/:userId", requireUser, h.View)
}
