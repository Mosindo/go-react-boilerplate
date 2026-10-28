package photos

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func asMaxBytes(err error, target **http.MaxBytesError) bool { return errors.As(err, target) }

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload := middleware.RateLimitByUser(middleware.NewLimiter(30, time.Hour))
	read := middleware.RateLimitByUser(middleware.NewLimiter(1200, time.Minute))

	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/file", requireUser, read, h.File)
}
