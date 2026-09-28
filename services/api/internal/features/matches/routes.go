package matches

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/matches", requireUser, h.List)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
