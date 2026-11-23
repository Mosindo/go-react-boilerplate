package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/profile", requireUser, h.GetMine)
	r.PATCH("/profile", requireUser, h.UpdateMine)
	r.PUT("/profile/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/profile/settings", requireUser, h.UpdateSettings)
	r.PUT("/profile/location", requireUser, h.UpdateLocation)
	r.DELETE("/profile/location", requireUser, h.ClearLocation)
	r.GET("/profiles/:userId", requireUser, h.GetPublic)
}
