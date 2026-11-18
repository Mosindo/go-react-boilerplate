package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, locationLimit gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.ListInterests)
	r.GET("/me/profile", requireUser, handler.GetMe)
	r.PUT("/me/profile", requireUser, handler.UpsertMe)
	r.PUT("/me/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/me/privacy", requireUser, handler.UpdatePrivacy)
	r.PUT("/me/location", requireUser, locationLimit, handler.UpdateLocation)
	r.DELETE("/me/location", requireUser, handler.ClearLocation)
	r.PUT("/me/interests", requireUser, handler.ReplaceInterests)
}
