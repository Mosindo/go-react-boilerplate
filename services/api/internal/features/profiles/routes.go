package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.Me)
	r.PUT("/me/profile", requireUser, h.UpdateProfile)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/me/location", requireUser, h.UpdateLocation)
	r.PUT("/me/privacy", requireUser, h.UpdatePrivacy)
	r.GET("/profiles/:userId", requireUser, h.Get)
}
