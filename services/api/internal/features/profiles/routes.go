package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile", requireUser, h.Self)
	r.PUT("/profile", requireUser, h.SaveProfile)
	r.PUT("/profile/privacy", requireUser, h.SavePrivacy)
	r.PUT("/profile/location", requireUser, h.SaveLocation)
	r.PUT("/profile/interests", requireUser, h.SaveInterests)
	r.PUT("/preferences", requireUser, h.SavePreferences)
	r.GET("/interests", requireUser, h.Catalog)
	r.GET("/profiles/:userId", requireUser, h.View)
}
