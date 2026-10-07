package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profile/me", requireUser, h.GetMe)
	r.PUT("/profile/me", requireUser, h.PutMe)
	r.PUT("/profile/me/location", requireUser, h.PutLocation)
	r.GET("/preferences", requireUser, h.GetPreferences)
	r.PUT("/preferences", requireUser, h.PutPreferences)
}
