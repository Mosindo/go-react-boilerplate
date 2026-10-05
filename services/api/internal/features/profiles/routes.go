package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile/me", requireUser, h.Mine)
	r.PUT("/profile/me", requireUser, h.Upsert)
	r.PUT("/profile/preferences", requireUser, h.SetPreferences)
	r.PUT("/profile/location", requireUser, h.SetLocation)
	r.PUT("/profile/privacy", requireUser, h.SetPrivacy)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profiles/:userId", requireUser, h.View)
}
