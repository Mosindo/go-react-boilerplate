package photos

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload := ratelimit.ByUser(ratelimit.New(20, 10))
	r.GET("/photos/:id/file", h.File)
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, upload, h.Replace)
	r.PUT("/photos/:id/primary", requireUser, h.MakePrimary)
	r.DELETE("/photos/:id", requireUser, h.Delete)
}
