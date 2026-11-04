package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/profile/me", h.Me)
	g.PUT("/profile/me", h.Upsert)
	g.PUT("/profile/preferences", h.Preferences)
	g.PUT("/profile/location", h.Location)
	g.GET("/interests", h.Interests)
	g.GET("/profiles/:id", h.Public)
}
