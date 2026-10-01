package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload := middleware.NewLimiter(30, time.Hour).ByUser()
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, upload, h.Add)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, upload, h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/image", requireUser, h.Image)
	r.GET("/photos/:id/thumb", requireUser, h.Thumb)
}
