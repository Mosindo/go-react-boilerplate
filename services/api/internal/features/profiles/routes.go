package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, locationRL gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/me/profile", requireUser, h.GetMine)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PUT("/me/location", requireUser, locationRL, h.UpdateLocation)
	r.PUT("/me/privacy", requireUser, h.UpdatePrivacy)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/me/interests", requireUser, h.ReplaceInterests)
	r.GET("/profiles/:userId", requireUser, h.GetCard)
}
