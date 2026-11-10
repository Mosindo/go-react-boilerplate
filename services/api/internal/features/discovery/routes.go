package discovery

import "github.com/gin-gonic/gin"

// swipeLimit is an anti-automation guard only (generous, never a quota): the
// product has no artificial like limits.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, swipeLimit gin.HandlerFunc) {
	g := r.Group("/discovery", requireUser)
	g.GET("", h.Deck)
	g.POST("/swipes", swipeLimit, h.Swipe)
}
