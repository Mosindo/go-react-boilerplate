package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.GetMine)
	r.PUT("/me/profile", requireUser, h.SaveMine)
	r.PUT("/me/preferences", requireUser, h.SavePreferences)
	r.PUT("/me/location", requireUser, h.SetLocation)
	r.GET("/profiles/:id", requireUser, h.GetPublic)
}
