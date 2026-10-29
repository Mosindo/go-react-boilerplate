package discovery

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, swipeLimit gin.HandlerFunc) {
	if swipeLimit == nil {
		swipeLimit = func(c *gin.Context) { c.Next() }
	}
	r.GET("/discover", requireUser, handler.Discover)
	r.POST("/swipes", requireUser, swipeLimit, handler.Swipe)
	r.DELETE("/matches/:id", requireUser, handler.Unmatch)
}
