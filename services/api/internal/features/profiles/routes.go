package profiles

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Location updates are throttled: frequent re-positioning is the basis of trilateration attacks.
	location := middleware.RateLimitByUser(middleware.NewLimiter(20, time.Hour))
	browse := middleware.RateLimitByUser(middleware.NewLimiter(240, time.Minute))

	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profile", requireUser, h.Get)
	r.PUT("/profile", requireUser, h.Save)
	r.PUT("/location", requireUser, location, h.SaveLocation)
	r.GET("/preferences", requireUser, h.GetPreferences)
	r.PUT("/preferences", requireUser, h.SavePreferences)
	r.GET("/profiles/:userId", requireUser, browse, h.Public)
}
