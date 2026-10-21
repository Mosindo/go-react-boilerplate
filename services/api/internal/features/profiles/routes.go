package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profile", requireUser, h.GetOwn)
	r.PUT("/profile", requireUser, h.Save)
	r.GET("/preferences", requireUser, h.GetPreferences)
	r.PUT("/preferences", requireUser, h.SavePreferences)
	r.GET("/users/:id", requireUser, h.View)
}
