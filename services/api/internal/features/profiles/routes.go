package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/interests", h.Interests)
	g.GET("/profile", h.GetOwn)
	g.PUT("/profile", h.UpsertOwn)
	g.PUT("/profile/preferences", h.UpdatePreferences)
	g.PUT("/profile/location", h.UpdateLocation)
	g.DELETE("/profile/location", h.ClearLocation)
	g.GET("/profiles/:userId", h.View)
}
