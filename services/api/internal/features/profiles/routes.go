package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", handler.Interests)
	r.GET("/me/profile", requireUser, handler.GetOwn)
	r.PUT("/me/profile", requireUser, handler.UpdateOwn)
	r.PUT("/me/location", requireUser, handler.UpdateLocation)
	r.DELETE("/me/location", requireUser, handler.ClearLocation)
	r.PUT("/me/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/me/privacy", requireUser, handler.UpdatePrivacy)
	r.GET("/profiles/:userId", requireUser, handler.GetVisible)
}
