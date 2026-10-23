package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/interests", h.Interests)
	g.GET("/me/profile", h.GetProfile)
	g.PUT("/me/profile", h.PutProfile)
	g.GET("/me/preferences", h.GetPreferences)
	g.PUT("/me/preferences", h.PutPreferences)
}
