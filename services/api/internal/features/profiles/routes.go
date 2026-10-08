package profiles

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	locationLimit := middleware.RateLimitByUser(middleware.NewLimiter(10, time.Hour))
	r.GET("/me", requireUser, h.Me)
	r.PUT("/me/profile", requireUser, h.UpdateProfile)
	r.PUT("/me/preferences", requireUser, h.UpdatePreferences)
	r.PUT("/me/location", requireUser, locationLimit, h.SetLocation)
	r.DELETE("/me/location", requireUser, h.ClearLocation)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profiles/:id", requireUser, h.View)
}
