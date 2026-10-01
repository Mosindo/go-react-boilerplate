package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/me/profile", requireUser, h.Get)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/me/location", requireUser, h.UpdateLocation)
	r.PUT("/me/interests", requireUser, h.SetInterests)
}
