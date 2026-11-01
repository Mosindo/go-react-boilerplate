package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/me/profile", requireUser, h.GetMyProfile)
	r.PUT("/me/profile", requireUser, h.PutMyProfile)
	r.GET("/me/settings", requireUser, h.GetSettings)
	r.PUT("/me/settings", requireUser, h.PutSettings)
	r.GET("/profiles/:id", requireUser, h.GetPublicProfile)
}
