package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.Interests)
	r.GET("/me/profile", requireUser, handler.Me)
	r.PATCH("/me/profile", requireUser, handler.Update)
	r.PUT("/me/location", requireUser, handler.UpdateLocation)
	r.PUT("/me/preferences", requireUser, handler.UpdatePreferences)
	r.PATCH("/me/settings", requireUser, handler.UpdateSettings)
	r.GET("/profiles/:userId", requireUser, handler.View)
}
