package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.ListInterests)
	r.GET("/me/profile", requireUser, handler.GetMine)
	r.PUT("/me/profile", requireUser, handler.UpsertMine)
	r.PUT("/me/preferences", requireUser, handler.UpdatePreferences)
	r.PUT("/me/interests", requireUser, handler.ReplaceInterests)
	r.PUT("/me/location", requireUser, handler.SetLocation)
	r.DELETE("/me/location", requireUser, handler.ClearLocation)
	r.PUT("/me/privacy", requireUser, handler.SetPrivacy)
	r.GET("/profiles/:userId", requireUser, handler.GetPublic)
}
