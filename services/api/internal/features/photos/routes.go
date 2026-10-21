package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	uploadLimit := middleware.NewRateLimiter(30, time.Hour).ByUser()

	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, uploadLimit, h.Upload)
	// Registered before /photos/:id so "order" is not parsed as an id.
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, uploadLimit, h.Replace)
	r.DELETE("/photos/:id", requireUser, h.Delete)

	// Signed URLs are the credential: no bearer token needed so image components can load them.
	r.GET("/media/photos/:id", h.Media)
}
