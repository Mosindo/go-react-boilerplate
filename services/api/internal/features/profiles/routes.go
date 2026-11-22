package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.GetOwn)
	r.PATCH("/me/profile", requireUser, h.Update)
	r.PUT("/me/location", requireUser, h.UpdateLocation)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.GET("/profiles/:userId", requireUser, h.Public)
}
