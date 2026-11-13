package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/me/profile", requireUser, handler.GetOwn)
	r.PATCH("/me/profile", requireUser, handler.Update)
	r.PUT("/me/location", requireUser, handler.UpdateLocation)
	r.PATCH("/me/preferences", requireUser, handler.UpdatePreferences)
	r.GET("/interests", requireUser, handler.Interests)
	r.GET("/profiles/:userId", requireUser, handler.GetPublic)
}
