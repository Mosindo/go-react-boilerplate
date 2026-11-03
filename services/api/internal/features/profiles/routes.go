package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.GetMine)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PUT("/me/location", requireUser, h.SetLocation)
	r.DELETE("/me/location", requireUser, h.ClearLocation)
	r.PUT("/me/privacy", requireUser, h.SetPrivacy)
	r.PUT("/me/preferences", requireUser, h.SetPreferences)
	r.GET("/profiles/:id", requireUser, h.GetCard)
}
