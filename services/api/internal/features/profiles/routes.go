package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/me/profile", requireUser, h.GetOwn)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PUT("/me/location", requireUser, h.SetLocation)
	r.PUT("/me/preferences", requireUser, h.SetPreferences)
	r.PUT("/me/privacy", requireUser, h.SetPrivacy)
	r.GET("/profiles/:userId", requireUser, h.GetPublic)
}
