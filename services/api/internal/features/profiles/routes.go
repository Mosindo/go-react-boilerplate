package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile", requireUser, h.GetOwn)
	r.PUT("/profile", requireUser, h.Save)
	r.PUT("/profile/location", requireUser, h.SetLocation)
	r.DELETE("/profile/location", requireUser, h.ClearLocation)
	r.GET("/preferences", requireUser, h.GetPreferences)
	r.PUT("/preferences", requireUser, h.SavePreferences)
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/profiles/:userId", requireUser, h.GetVisible)
}
