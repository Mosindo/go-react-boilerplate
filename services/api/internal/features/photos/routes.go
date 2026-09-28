package photos

import "github.com/gin-gonic/gin"

// RegisterRoutes takes two groups because uploads need a larger body limit than
// the JSON default: `api` carries the small JSON limit, `upload` the ~8.5 MB one.
func RegisterRoutes(api, upload gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload.POST("/me/photos", requireUser, h.Upload)
	upload.PUT("/me/photos/:id", requireUser, h.Replace)
	api.PUT("/me/photos/order", requireUser, h.Reorder)
	api.DELETE("/me/photos/:id", requireUser, h.Delete)
	api.GET("/photos/:id/content", requireUser, h.Content)
}
