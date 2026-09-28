package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/me", requireUser, h.Me)
	r.PUT("/me/profile", requireUser, h.UpdateProfile)
	r.PUT("/me/location", requireUser, h.UpdateLocation)
	r.GET("/me/preferences", requireUser, h.GetPreferences)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.GET("/interests", requireUser, h.Interests)
}
