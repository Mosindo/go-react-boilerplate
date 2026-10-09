package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/me", requireUser, h.Me)
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/me/profile", requireUser, h.GetProfile)
	r.PUT("/me/profile", requireUser, h.UpdateProfile)
	r.GET("/me/preferences", requireUser, h.GetPreferences)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.GET("/profiles/:userId", requireUser, h.PublicProfile)
}
