package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, locationLimit gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/profile", requireUser, h.GetOwn)
	r.PUT("/profile", requireUser, h.Upsert)
	r.PUT("/profile/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/profile/interests", requireUser, h.SetInterests)
	// Location updates are rate limited to make distance triangulation impractical.
	r.PUT("/profile/location", requireUser, locationLimit, h.UpdateLocation)
	r.DELETE("/profile/location", requireUser, h.ClearLocation)
	r.PUT("/profile/privacy", requireUser, h.UpdatePrivacy)
	r.GET("/profiles/:userId", requireUser, h.GetPublic)
}
