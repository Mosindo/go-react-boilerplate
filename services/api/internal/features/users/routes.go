package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.Interests)
	r.GET("/me/profile", requireUser, handler.Me)
	r.PUT("/me/profile", requireUser, handler.SaveProfile)
	r.PUT("/me/preferences", requireUser, handler.SavePreferences)
	r.PUT("/me/location", requireUser, handler.SetLocation)
	r.DELETE("/me/location", requireUser, handler.ClearLocation)
	r.PATCH("/me/privacy", requireUser, handler.SetPrivacy)
}
