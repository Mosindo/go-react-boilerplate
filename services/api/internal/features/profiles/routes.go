package profiles

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/interests", requireUser, handler.ListInterests)
	r.GET("/profile", requireUser, handler.GetProfile)
	r.PUT("/profile", requireUser, handler.PutProfile)
	r.GET("/profile/preferences", requireUser, handler.GetPreferences)
	r.PUT("/profile/preferences", requireUser, handler.PutPreferences)
	r.GET("/profiles/:id", requireUser, handler.View)
}
