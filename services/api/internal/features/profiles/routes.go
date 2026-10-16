package profiles

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	writeLimit := middleware.NewLimiter(60, time.Minute).ByUser()

	r.GET("/me/profile", requireUser, h.Me)
	r.PUT("/me/profile", requireUser, writeLimit, h.SaveProfile)
	r.PUT("/me/preferences", requireUser, writeLimit, h.SavePreferences)
	r.PATCH("/me/privacy", requireUser, writeLimit, h.SavePrivacy)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profiles/:userId", requireUser, h.View)
}
