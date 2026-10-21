package matches

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	// Anti-abuse ceiling (scripted mass-liking), far above human swiping speed. Not a product limit.
	swipeLimit := middleware.NewRateLimiter(300, time.Minute).ByUser()

	r.POST("/swipes", requireUser, swipeLimit, h.Swipe)
	r.DELETE("/swipes/passes", requireUser, h.ResetPasses)
	r.GET("/matches", requireUser, h.List)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
