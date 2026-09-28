package realtime

import "github.com/gin-gonic/gin"

// RegisterRoutes exposes GET /ws. Authentication happens through the first
// WebSocket message, not a header, so no RequireUser middleware here.
func RegisterRoutes(r gin.IRouter, h *Handler) {
	r.GET("/ws", h.Serve)
}
