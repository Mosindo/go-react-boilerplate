package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.ListInterests)
	r.GET("/profile", requireUser, handler.GetOwn)
	r.PATCH("/profile", requireUser, handler.Update)
	r.PUT("/profile/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/profile/interests", requireUser, handler.UpdateInterests)
	r.PUT("/profile/location", requireUser, handler.UpdateLocation)
	r.DELETE("/profile/location", requireUser, handler.ClearLocation)
	r.GET("/profiles/:userId", requireUser, handler.GetPublic)
}
