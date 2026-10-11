package users

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.GetOwnProfile)
	r.PUT("/me/profile", requireUser, h.PutProfile)
	r.PUT("/me/location", requireUser, h.PutLocation)
	r.DELETE("/me/location", requireUser, h.DeleteLocation)
	r.GET("/me/preferences", requireUser, h.GetPreferences)
	r.PUT("/me/preferences", requireUser, h.PutPreferences)
	r.GET("/me/privacy", requireUser, h.GetPrivacy)
	r.PUT("/me/privacy", requireUser, h.PutPrivacy)
	r.GET("/profiles/:userId", requireUser, h.GetProfile)
}
