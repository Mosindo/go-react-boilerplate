package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.ListInterests)
	r.GET("/profile/me", requireUser, handler.GetMine)
	r.PUT("/profile/me", requireUser, handler.UpsertMine)
	r.PUT("/profile/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/profile/location", requireUser, handler.UpdateLocation)
	r.GET("/profiles/:userId", requireUser, handler.GetProfile)
}
