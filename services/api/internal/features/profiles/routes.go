package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.Interests)
	r.GET("/profile", requireUser, handler.GetOwn)
	r.PUT("/profile", requireUser, handler.Upsert)
	r.PUT("/profile/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/profile/location", requireUser, handler.UpdateLocation)
	r.DELETE("/profile/location", requireUser, handler.ClearLocation)
	r.PUT("/profile/visibility", requireUser, handler.UpdateVisibility)
	r.GET("/profiles/:userId", requireUser, handler.View)
}
