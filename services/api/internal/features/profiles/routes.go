package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", h.Interests)
	r.GET("/me/profile", requireUser, h.GetOwn)
	r.PUT("/me/profile", requireUser, h.Update)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.GET("/profiles/:id", requireUser, h.Get)
}
