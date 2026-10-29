package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, uploadLimit gin.HandlerFunc) {
	if uploadLimit == nil {
		uploadLimit = func(c *gin.Context) { c.Next() }
	}
	r.GET("/photos", requireUser, handler.List)
	r.POST("/photos", requireUser, uploadLimit, handler.Upload)
	r.PUT("/photos/order", requireUser, handler.Reorder)
	r.DELETE("/photos/:id", requireUser, handler.Delete)
	r.GET("/photos/:id/file", requireUser, handler.File)
}
