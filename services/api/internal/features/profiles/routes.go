package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", h.Interests)
	me := r.Group("/me", requireUser)
	me.GET("/profile", h.Mine)
	me.PUT("/profile", h.Update)
	me.PUT("/preferences", h.UpdatePreferences)
	me.PUT("/location", h.UpdateLocation)
	me.DELETE("/location", h.ClearLocation)
	me.PUT("/privacy", h.UpdatePrivacy)
	r.GET("/profiles/:userId", requireUser, h.View)
}
