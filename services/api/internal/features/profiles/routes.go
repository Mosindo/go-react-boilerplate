package profiles

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	write := limiter.ByUser("profile-write", 60, time.Minute)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profile", requireUser, h.Own)
	r.PUT("/profile", requireUser, write, h.Save)
	r.PUT("/profile/preferences", requireUser, write, h.SavePreferences)
	// Tight limit: repeated location changes combined with distance readings would let someone triangulate a member.
	r.PUT("/profile/location", requireUser, limiter.ByUser("profile-location", 10, time.Hour), h.SetLocation)
	r.GET("/profiles/:id", requireUser, limiter.ByUser("profile-read", 300, time.Minute), h.Public)
}
