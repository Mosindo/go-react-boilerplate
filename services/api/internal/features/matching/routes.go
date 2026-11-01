package matching

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.POST("/profiles/:id/like", requireUser, swipeLimit, h.swipe(ActionLike))
	r.POST("/profiles/:id/pass", requireUser, swipeLimit, h.swipe(ActionPass))
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
