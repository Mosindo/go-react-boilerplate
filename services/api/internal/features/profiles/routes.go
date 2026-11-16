package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, locationLimit gin.HandlerFunc) {
	r.GET("/interests", h.Interests)
	r.GET("/me/profile", requireUser, h.Me)
	r.PUT("/me/profile", requireUser, h.SaveProfile)
	r.PUT("/me/preferences", requireUser, h.SavePreferences)
	r.PUT("/me/location", requireUser, locationLimit, h.UpdateLocation)
	r.PUT("/me/visibility", requireUser, h.SetVisibility)
	r.GET("/profiles/:userId", requireUser, h.View)
}
