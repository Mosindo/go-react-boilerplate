package profiles

import "github.com/gin-gonic/gin"

// locationLimit slows down position changes, which would otherwise allow distance probing.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, locationLimit gin.HandlerFunc) {
	r.GET("/interests", h.ListInterests)

	g := r.Group("", requireUser)
	g.GET("/profile", h.GetMine)
	g.PUT("/profile", h.UpdateMine)
	g.PUT("/profile/preferences", h.UpdatePreferences)
	g.PUT("/profile/location", locationLimit, h.UpdateLocation)
	g.DELETE("/profile/location", h.ClearLocation)
	g.PATCH("/profile/privacy", h.UpdatePrivacy)
	g.GET("/profiles/:userId", h.GetPublic)
}
