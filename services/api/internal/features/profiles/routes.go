package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.Interests)
	r.GET("/me/profile", requireUser, handler.Me)
	r.PUT("/me/profile", requireUser, handler.UpsertProfile)
	r.PUT("/me/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/me/interests", requireUser, handler.UpdateInterests)
	r.PUT("/me/location", requireUser, handler.UpdateLocation)
	r.DELETE("/me/location", requireUser, handler.ClearLocation)
	r.PATCH("/me/settings", requireUser, handler.UpdateSettings)
	r.GET("/profiles/:userId", requireUser, handler.View)
}
