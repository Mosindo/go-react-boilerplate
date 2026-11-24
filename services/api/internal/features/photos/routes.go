package photos

import "github.com/gin-gonic/gin"

type Guards struct {
	RequireUser gin.HandlerFunc
	UploadRL    gin.HandlerFunc
	UploadBody  gin.HandlerFunc
	JSONBody    gin.HandlerFunc
}

func RegisterRoutes(r gin.IRouter, h *Handler, g Guards) {
	r.GET("/me/photos", g.RequireUser, h.List)
	r.POST("/me/photos", g.RequireUser, g.UploadRL, g.UploadBody, h.Upload)
	r.PUT("/me/photos/order", g.RequireUser, g.JSONBody, h.Reorder)
	r.PUT("/me/photos/:photoId", g.RequireUser, g.UploadRL, g.UploadBody, h.Replace)
	r.DELETE("/me/photos/:photoId", g.RequireUser, h.Delete)
	r.GET("/media/photos/:photoId", h.Serve)
}
