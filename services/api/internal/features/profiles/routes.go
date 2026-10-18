package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/me/profile", requireUser, h.Get)
	r.PUT("/me/profile", requireUser, h.Upsert)
	r.PATCH("/me/settings", requireUser, h.UpdateSettings)
}
