package matching

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeLimiter *middleware.RateLimiter) {
	r.GET("/discover", requireUser, h.Feed)
	r.GET("/likes/received", requireUser, h.ReceivedLikes)
	r.POST("/discover/:id/like", requireUser, swipeLimiter.ByUser(), h.Like)
	r.POST("/discover/:id/pass", requireUser, swipeLimiter.ByUser(), h.Pass)
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
