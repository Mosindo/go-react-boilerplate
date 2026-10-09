package photos

import (
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	uploads := ratelimit.ByUser("photo-upload", ratelimit.New(30, time.Hour))

	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploads, h.Upload)
	// "order" is registered before the :id routes; Gin resolves static segments first.
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, uploads, h.Replace)
	r.POST("/me/photos/:id/main", requireUser, h.SetMain)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/file", requireUser, h.File)
}
