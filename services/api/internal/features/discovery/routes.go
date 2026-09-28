package discovery

import "github.com/gin-gonic/gin"

// RegisterRoutes wires discovery. swipeMW (per-user rate limit) applies to
// the swipe endpoints only.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, swipeMW ...gin.HandlerFunc) {
	r.GET("/discover", requireUser, h.Discover)
	r.GET("/users/:id/profile", requireUser, h.Profile)
	swipe := append([]gin.HandlerFunc{requireUser}, swipeMW...)
	r.POST("/swipes", append(swipe, h.Swipe)...)
	r.DELETE("/swipes/last", requireUser, h.UndoNotImplemented)
}
