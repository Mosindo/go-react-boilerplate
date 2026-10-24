package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.Own)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PUT("/me/preferences", requireUser, h.SetPreferences)
	r.PUT("/me/location", requireUser, h.SetLocation)
	r.DELETE("/me/location", requireUser, h.ClearLocation)
	r.GET("/profiles/:userId", requireUser, h.Public)
}
