package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	uploadLimit := middleware.NewLimiter(30, time.Hour).ByUser()
	editLimit := middleware.NewLimiter(120, time.Hour).ByUser()

	r.POST("/me/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/me/photos/order", requireUser, editLimit, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploadLimit, h.Replace)
	r.PUT("/me/photos/:photoId/main", requireUser, editLimit, h.MakeMain)
	r.DELETE("/me/photos/:photoId", requireUser, editLimit, h.Delete)
	r.GET("/media/photos/:photoId", h.Serve)
}
