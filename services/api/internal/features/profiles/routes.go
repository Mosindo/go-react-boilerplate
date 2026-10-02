package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/profile", requireUser, h.Get)
	r.PUT("/profile", requireUser, h.Put)
	r.PUT("/preferences", requireUser, h.PutPreferences)
	r.GET("/interests", requireUser, h.Interests)
}
