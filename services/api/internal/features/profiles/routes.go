package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile", requireUser, h.GetMine)
	r.PUT("/profile", requireUser, h.UpdateMine)
	r.GET("/preferences", requireUser, h.GetPreferences)
	r.PUT("/preferences", requireUser, h.UpdatePreferences)
	r.GET("/interests", requireUser, h.ListInterests)
	r.GET("/profiles/:userId", requireUser, h.GetPublic)
}
