package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile", requireUser, h.Get)
	r.PUT("/profile", requireUser, h.Upsert)
	r.PUT("/preferences", requireUser, h.SetPreferences)
	r.PUT("/profile/location", requireUser, h.SetLocation)
	r.DELETE("/profile/location", requireUser, h.ClearLocation)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profiles/:userId", requireUser, h.View)
}
