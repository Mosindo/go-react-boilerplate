package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Catalog)
	r.GET("/me/profile", requireUser, h.GetMine)
	r.PUT("/me/profile", requireUser, h.SaveProfile)
	r.PUT("/me/preferences", requireUser, h.SavePreferences)
	r.PUT("/me/interests", requireUser, h.SaveInterests)
	r.PUT("/me/location", requireUser, h.SetLocation)
	r.DELETE("/me/location", requireUser, h.ClearLocation)
	r.PUT("/me/visibility", requireUser, h.SetVisibility)
	r.GET("/profiles/:userId", requireUser, h.View)
}
